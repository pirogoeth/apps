#!/usr/bin/env bash

this_dir="$(dirname $(readlink -nf "${BASH_SOURCE[0]}"))"
echo "this_dir: ${this_dir}"
mibs_dir="$(readlink -nf "${this_dir}/../snmpsmi/mibs")"
echo "mibs_dir: ${mibs_dir}"
mkdir -p "${mibs_dir}"

NO_CLEAN="${NO_CLEAN:-0}"
VERSION=5.9.4

tmpdir=$(mktemp -p /tmp -d maparoon-build.XXXX)
echo "tmpdir: ${tmpdir}"

pushd "${tmpdir}"

echo "Downloading net-snmp-${VERSION}.zip"
curl -L -o net-snmp.zip https://sourceforge.net/projects/net-snmp/files/net-snmp/${VERSION}/net-snmp-${VERSION}.zip/download
unzip -q net-snmp.zip
cd net-snmp-${VERSION}/mibs
cp -rv ./*.txt "${mibs_dir}"

popd

if [[ "${NO_CLEAN}" -eq "0" ]]; then
    rm -r "${tmpdir}"
fi