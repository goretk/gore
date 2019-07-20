#!/usr/bin/env bash

BUILD_FOLDER="/tmp/gore_testdata"
TESTDATA_FOLDER=$(dirname $0)
REPO_LOCATION="/keybase/public/joakimkennedy/gore-test"
REPO_URL="https://joakimkennedy.keybase.pub/gore-test"

# Load vars
source $TESTDATA_FOLDER/versions
source $TESTDATA_FOLDER/vars

function fetch() {
    if [ -f $REPO_LOCATION/latest ]; then
        LATEST_TAR=$(cat $REPO_LOCATION/latest)
        tar xfvj $REPO_LOCATION/$LATEST_TAR -C $TESTDATA_FOLDER
    else
       LATEST_TAR=$(wget -qO- $REPO_URL/latest 2> /dev/null || curl -s $REPO_URL/latest)
       (wget -qO- $REPO_URL/$LATEST_TAR 2> /dev/null || curl -s $REPO_URL/$LATEST_TAR) | tar xvjf - -C $TESTDATA_FOLDER
    fi
}

function create_tar() {
    tar cfjv gore-golden-$(date -u +"%Y%m%d%H%M").tar.bz2 -C $TESTDATA_FOLDER gold
}

function upload_and_clean() {
    NEW_TAR=$(ls gore-golden-*.tar.bz2)
    cp -v $NEW_TAR $REPO_LOCATION/.
    echo "Removing archive"
    rm $NEW_TAR
    echo "Update latest file record"
    echo "$NEW_TAR" > $REPO_LOCATION/latest
}

function build_in_container() {
   docker run --rm -it -e GOOS=$2 -e GOARCH=$3 -v $BUILD_FOLDER:/build golang:"$1" go build -ldflags="-s -w" -o /build/$4 /build/target.go
}

function build_testdata() {
   mkdir -p $BUILD_FOLDER
   cp $TESTDATA_FOLDER/simple.go $BUILD_FOLDER/target.go
   for version in "${AVAILABLE_VERSIONS[@]}"; do
      for os in "${AVAILABLE_OS[@]}"; do
         for arch in "${AVAILABLE_ARCH[@]}"; do
            FILE="gold-$os-$arch-$version"
            if [ ! -f $TESTDATA_FOLDER/gold/$FILE ]; then
               echo "Building $FILE"
               build_in_container $version $os $arch $FILE
            fi
         done
      done
   done

    # Move files to folder
    mkdir -p $TESTDATA_FOLDER/gold
    mv $BUILD_FOLDER/* $TESTDATA_FOLDER/gold/.

    # Clean up.
    rm $TESTDATA_FOLDER/gold/target.go
    rm -r $BUILD_FOLDER
 }

case "$1" in
   build)
      build_testdata
      ;;
   upload)
      create_tar
      upload_and_clean
      ;;
   fetch)
      fetch
      ;;
   *)
      echo "$1 Didn't match anything"
esac

