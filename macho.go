package gore

import (
	"debug/gosym"
	"debug/macho"
	"fmt"
)

func openMachO(fp string) (*machoFile, error) {
	f, err := macho.Open(fp)
	if err != nil {
		return nil, err
	}
	return &machoFile{file: f}, nil
}

type machoFile struct {
	file *macho.File
}

func (m *machoFile) Close() error {
	return m.file.Close()
}

func (m *machoFile) getPCLNTab() (*gosym.Table, error) {
	section := m.file.Section("__gopclntab")
	if section == nil {
		return nil, ErrNoPCLNTab
	}
	data, err := section.Data()
	if data == nil {
		return nil, err
	}
	pcln := gosym.NewLineTable(data, m.file.Section("__text").Addr)
	return gosym.NewTable(nil, pcln)
}

func (m *machoFile) getRData() ([]byte, error) {
	_, data, err := m.getSectionData("__rodata")
	return data, err
}

func (m *machoFile) getCodeSection() ([]byte, error) {
	_, data, err := m.getSectionData("__text")
	return data, err
}

func (m *machoFile) getSectionDataFromOffset(off uint64) (uint64, []byte, error) {
	for _, section := range m.file.Sections {
		if section.Addr <= off && off < (section.Addr+section.Size) {
			data, err := section.Data()
			return section.Addr, data, err
		}
	}
	return 0, nil, ErrSectionDoesNotExist
}

func (m *machoFile) getSectionData(s string) (uint64, []byte, error) {
	section := m.file.Section(s)
	if section == nil {
		return 0, nil, ErrSectionDoesNotExist
	}
	data, err := section.Data()
	return section.Addr, data, err
}

func (m *machoFile) getFileInfo() *FileInfo {
	var wordSize int
	switch m.file.Cpu {
	case macho.Cpu386:
		wordSize = intSize32
	case macho.CpuAmd64:
		wordSize = intSize64
	default:
		panic("Unsupported architecture")
	}
	return &FileInfo{
		ByteOrder: m.file.ByteOrder,
		OS:        "macOS",
		WordSize:  wordSize,
	}
}

func (m *machoFile) getPCLNTABData() (uint64, []byte, error) {
	return m.getSectionData("__gopclntab")
}

func (m *machoFile) moduledataSection() string {
	return "__noptrdata"
}

func (m *machoFile) getBuildID() (string, error) {
	data, err := m.getCodeSection()
	if err != nil {
		return "", fmt.Errorf("failed to get code section: %w", err)
	}
	return parseBuildIDFromRaw(data)
}
