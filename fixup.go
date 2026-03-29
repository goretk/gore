package gore

import (
	"debug/elf"
	"encoding/binary"

	"github.com/blacktop/go-macho"
)

// pointerResolver resolves relocated pointer values in PIE binaries.
// Implementations handle Mach-O chained fixups and ELF RELATIVE relocations.
type pointerResolver interface {
	// ResolvePointer returns the relocated value of a pointer stored at fileAddr.
	// Non-pointer values are returned unchanged.
	ResolvePointer(val uint64, fileAddr uint64) uint64
}

// newPointerResolver returns a resolver for the given file handler,
// or nil if the binary has no pointer relocations to resolve.
func newPointerResolver(fh fileHandler) pointerResolver {
	if fh == nil {
		return nil
	}
	switch f := fh.getParsedFile().(type) {
	case *macho.File:
		if fh.getFileInfo().WordSize != 8 {
			return nil
		}
		return newMachoResolver(f)
	case *elf.File:
		if r := newElfResolver(f, fh.getFileInfo()); r != nil {
			return r
		}
		return nil
	}
	return nil
}

// resolvePointer is a nil-safe helper that calls r.ResolvePointer if r is non-nil.
func resolvePointer(r pointerResolver, val uint64, fileAddr uint64) uint64 {
	if r == nil {
		return val
	}
	return r.ResolvePointer(val, fileAddr)
}

// resolveBuffer resolves all word-sized slots in data where the resolver
// unambiguously changes the value (i.e., raw != resolved). Used for moduledata
// where fields mix pointers and non-pointer integers. Returns a new buffer.
func resolveBuffer(r *machoResolver, data []byte, baseAddr uint64, wordSize int, order binary.ByteOrder) []byte {
	if r == nil {
		return data
	}
	resolved := make([]byte, len(data))
	copy(resolved, data)
	for i := 0; i+wordSize <= len(data); i += wordSize {
		var raw uint64
		if wordSize == 4 {
			raw = uint64(order.Uint32(data[i:]))
		} else {
			raw = order.Uint64(data[i:])
		}
		val, err := r.mf.GetSlidPointerAtAddress(baseAddr + uint64(i))
		if err != nil || val == raw {
			continue
		}
		if val < r.imageBase {
			val += r.imageBase
		}
		if wordSize == 4 {
			order.PutUint32(resolved[i:], uint32(val))
		} else {
			order.PutUint64(resolved[i:], val)
		}
	}
	return resolved
}

// findPointerValue scans word-sized slots looking for one resolving to targetAddr.
func findPointerValue(r pointerResolver, secAddr uint64, secData []byte, targetAddr uint64, wordSize int, order binary.ByteOrder) int {
	if r == nil {
		return -1
	}
	for i := 0; i+wordSize <= len(secData); i += wordSize {
		var raw uint64
		if wordSize == 4 {
			raw = uint64(order.Uint32(secData[i:]))
		} else {
			raw = order.Uint64(secData[i:])
		}
		if r.ResolvePointer(raw, secAddr+uint64(i)) == targetAddr {
			return i
		}
	}
	return -1
}

// --- Mach-O implementation ---

type machoResolver struct {
	mf        *macho.File
	imageBase uint64
	chained   bool
}

func newMachoResolver(mf *macho.File) *machoResolver {
	return &machoResolver{
		mf:        mf,
		imageBase: machoImageBase(mf),
		chained:   mf.HasDyldChainedFixups(),
	}
}

func (r *machoResolver) ResolvePointer(val uint64, fileAddr uint64) uint64 {
	if val == 0 {
		return val
	}
	resolved, err := r.mf.GetSlidPointerAtAddress(fileAddr)
	if err != nil {
		return val
	}
	if resolved != val {
		if resolved < r.imageBase {
			resolved += r.imageBase
		}
		return resolved
	}
	// Chain-tail pointer: resolved == raw but may lack image base in chained fixup binaries.
	if r.chained && resolved < r.imageBase && r.looksLikePointer(resolved) {
		return resolved + r.imageBase
	}
	return val
}

func (r *machoResolver) looksLikePointer(val uint64) bool {
	target := val + r.imageBase
	for _, seg := range r.mf.Segments() {
		if seg.Memsz > 0 && target >= seg.Addr && target < seg.Addr+seg.Memsz {
			return true
		}
	}
	return false
}

func machoImageBase(mf *macho.File) uint64 {
	for _, seg := range mf.Segments() {
		if seg.Name == "__TEXT" {
			return seg.Addr
		}
	}
	return 0
}

// --- ELF implementation ---

// elfResolver resolves R_*_RELATIVE relocations using a prebuilt offset→addend map.
type elfResolver struct {
	relocs map[uint64]uint64 // file offset → resolved address (addend)
}

func newElfResolver(f *elf.File, fi *FileInfo) *elfResolver {
	relocs := make(map[uint64]uint64)
	is32 := fi.WordSize == 4

	for _, s := range f.Sections {
		if is32 && s.Type == elf.SHT_REL {
			data, err := s.Data()
			if err != nil {
				continue
			}
			// 32-bit REL: 4-byte offset + 4-byte info, no addend.
			// R_386_RELATIVE: final addr = base + *(offset). For static
			// analysis base=0, so the value at the offset IS the address.
			// We skip these since the file already contains the right value
			// at base=0... actually no, the value in the file is the addend
			// for REL entries. We need to just mark these offsets.
			relType386Relative := uint32(8) // R_386_RELATIVE
			for i := 0; i+8 <= len(data); i += 8 {
				offset := uint64(fi.ByteOrder.Uint32(data[i:]))
				info := fi.ByteOrder.Uint32(data[i+4:])
				if info == relType386Relative {
					// For R_386_RELATIVE with REL (no explicit addend),
					// the addend is stored at the relocation target in the file.
					// We read it from the file via getSectionDataFromAddress later.
					// For now, mark with 0 as sentinel — we handle this in ResolvePointer.
					relocs[offset] = 0
				}
			}
		}

		if s.Type == elf.SHT_RELA {
			data, err := s.Data()
			if err != nil {
				continue
			}
			if is32 {
				relType386Relative := uint32(8) // R_386_RELATIVE
				for i := 0; i+12 <= len(data); i += 12 {
					offset := uint64(fi.ByteOrder.Uint32(data[i:]))
					info := fi.ByteOrder.Uint32(data[i+4:])
					addend := uint64(int64(int32(fi.ByteOrder.Uint32(data[i+8:]))))
					if info == relType386Relative {
						relocs[offset] = addend
					}
				}
			} else {
				// 64-bit RELA: 8-byte offset + 8-byte info + 8-byte addend
				var relTypeRelative uint32
				switch f.Machine {
				case elf.EM_X86_64:
					relTypeRelative = uint32(elf.R_X86_64_RELATIVE)
				case elf.EM_AARCH64:
					relTypeRelative = uint32(elf.R_AARCH64_RELATIVE)
				case elf.EM_386:
					relTypeRelative = uint32(elf.R_386_RELATIVE)
				default:
					continue
				}
				for i := 0; i+24 <= len(data); i += 24 {
					offset := fi.ByteOrder.Uint64(data[i:])
					info := fi.ByteOrder.Uint64(data[i+8:])
					addend := fi.ByteOrder.Uint64(data[i+16:])
					if uint32(info&0xffffffff) == relTypeRelative {
						relocs[offset] = addend
					}
				}
			}
		}
	}

	if len(relocs) == 0 {
		return nil
	}
	return &elfResolver{relocs: relocs}
}

func (r *elfResolver) ResolvePointer(val uint64, fileAddr uint64) uint64 {
	if addend, ok := r.relocs[fileAddr]; ok {
		if addend != 0 {
			return addend
		}
		// REL entry (no explicit addend): the value in the file IS the addend.
		return val
	}
	return val
}
