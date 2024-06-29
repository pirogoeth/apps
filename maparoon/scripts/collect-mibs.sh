#!/usr/bin/env bash

echo
echo " ******************************************************* "
echo " ***                                                 *** "
echo " *** Hey! This script collects MIB files from        *** "
echo " *** the internet. Check out the source here         *** "
echo " *** to see what's going on: scripts/collect-mibs.sh *** "
echo " ***                                                 *** "
echo " ******************************************************* "
echo
sleep 3


set -e

NO_CLEAN="${NO_CLEAN:-0}"
NETSNMP_VERSION=5.9.4
JUNIPER_MIBS_URL="https://www.juniper.net/documentation/software/junos/junos234/juniper-mibs-23.4R1.10.zip"

this_dir="$(dirname $(readlink -nf "${BASH_SOURCE[0]}"))"
echo "this_dir: ${this_dir}"

tmpdir=$(mktemp -p /tmp -d maparoon-build.XXXX)
echo "tmpdir: ${tmpdir}"

pushd "${tmpdir}"

# ------------------------------------------------------------------

echo
echo " ********************************************** "
echo " ***                                        *** "
echo " *** Collecting standard MIBs from net-snmp *** "
echo " ***                                        *** "
echo " ********************************************** "
echo

mibs_dir="$(readlink -nf "${this_dir}/../snmpsmi/netSnmpMibs")"
echo "mibs_dir: ${mibs_dir}"
mkdir -p "${mibs_dir}"

echo "Downloading net-snmp-${NETSNMP_VERSION}.zip"
curl -L -o net-snmp.zip https://sourceforge.net/projects/net-snmp/files/net-snmp/${NETSNMP_VERSION}/net-snmp-${NETSNMP_VERSION}.zip/download
unzip -q net-snmp.zip
cd net-snmp-${NETSNMP_VERSION}/mibs
cp -rv ./*.txt "${mibs_dir}"

# ------------------------------------------------------------------

echo
echo " ********************************************************** "
echo " ***                                                    *** "
echo " *** Collecting Juniper MIBs from Juniper SNMP Explorer *** "
echo " ***                                                    *** "
echo " ********************************************************** "
echo


mibs_dir="$(readlink -nf "${this_dir}/../snmpsmi/juniperMibs")"
echo "mibs_dir: ${mibs_dir}"
mkdir -p "${mibs_dir}"

echo "Downloading Juniper MIBs from ${JUNIPER_MIBS_URL}"
curl -L -o juniper-mibs.zip "${JUNIPER_MIBS_URL}"
unzip -q juniper-mibs.zip
cp -rv ./JuniperMibs/*.txt "${mibs_dir}"

# ------------------------------------------------------------------

popd

if [[ "${NO_CLEAN}" -eq "0" ]]; then
    rm -r "${tmpdir}"
fi
