# Copyright 2020 Megaport Pty Ltd
#
# Licensed under the Mozilla Public License, Version 2.0 (the
# "License"); you may not use this file except in compliance with
# the License. You may obtain a copy of the License at
#
#       https://mozilla.org/MPL/2.0/
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

provider_directory="$(pwd)"
GO111MODULE=on GOSUMDB=off go get -d github.com/megaport/megaportgo
rm -f bin/*
version="$(git describe --tags)"
provider_filename="$(pwd)/bin/terraform-provider-megaport_$version"
provider_filename_no_version="$(pwd)/bin/terraform-provider-megaport"
go build -o $provider_filename
cp $provider_filename "bin/terraform-provider-megaport"
chmod +x bin/terraform-provider-megaport*
echo "Provider built at '${provider_filename}'."
cd ~
arch=$(go version | cut -d" " -f4 | sed 's/\//_/g')
plugin_directory="$(pwd)/.terraform.d/plugins/${arch}/"
mkdir -p $plugin_directory
ln -s $provider_filename_no_version $plugin_directory
echo "Symbolic link created from build directory to terraform.d. < 0.13"
plugin_directory="$(pwd)/.terraform.d/plugins/megaport.com/megaport/megaport/${version:1}/${arch}/"
mkdir -p $plugin_directory
ln -s $provider_filename_no_version $plugin_directory
echo "Symbolic link created from build directory to terraform.d. >= 0.13"
cd $provider_directory
