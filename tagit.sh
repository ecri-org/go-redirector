#!/usr/bin/env bash

export versionFile="./.version"
export DO_TAG=${DO_TAG:-"false"}

getVer() { echo "$(cat "${versionFile}" | awk '{$1=$1};1')"; }
incVer() { echo "$(bash ./semver-inc.sh -m $(getVer))"; }

[ ! -e "${versionFile}" ] && echo "0.0.0" > "${versionFile}";

if [ "$DO_TAG"=="true" ]; then
	echo "$(incVer)" > "${versionFile}"
	echo "version tagged as: $(getVer)"
else
	echo "New version _would_ be tagged as $(incVer)"
fi
