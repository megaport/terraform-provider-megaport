/**
 * Copyright 2020 Megaport Pty Ltd
 *
 * Licensed under the Mozilla Public License, Version 2.0 (the
 * "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *       https://mozilla.org/MPL/2.0/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

output "aws_instance_ip" {
  value = aws_instance.tf_test.private_ip
}

output "gcp_instance_ip" {
  value = google_compute_instance.tf_test.network_interface.0.network_ip
}

output "ssh_command" {
  value = format("ssh -i ~/.ssh/%s	ec2-user@%s", aws_instance.tf_test.key_name, aws_instance.tf_test.private_ip)
}
