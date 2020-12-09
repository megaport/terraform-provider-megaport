This examples provisions a full multi-cloud demonstration environment including networking and compute instances. It 
requires account credentials for Megaport, Amazon Web Services, and Azure.

This example requires some prior understanding of AWS and Azure platforms, as well as usage of SSH and key pairs.  

## Before you begin
  * Complete the [Getting Started Requirements](https://registry.terraform.io/providers/megaport/megaport/latest/docs/guides/gettingstarted)
  * You will also need to authenticate to Azure using one of the methods supported by the Azure
    Resource Manager Provider. A full list of these are available 
    [Authentication to Azure](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs).  
  * You will also need valid AWS credentials in your OS shell. The 
    [AWS CLI documentation](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html) details 
    the methods for configuring AWS Credentials.  
  * Create a key-pair in AWS for the compute instance and update the `aws_ec2_key_pair_name` in `variables.tf`,
    the default key-pair name is `terraform-testing`

## Notes
This example will not work on the Megaport Staging environment. This is because it requires
real connections to AWS and Azure.

When you have completed, use `terraform destroy` so that you will stop incurring costs for the resources.

This example will create an Azure Bastion host which will allow you to logon to the Azure Portal using the
bastion functionality.

## Testing

Once Terraform has completed the build it will output the IP addresses of the compute resources, similar to:
```
aws_instance_ip = 172.16.1.1
ssh_command = ssh -i ~/.ssh/terraform-testing        ec2-user@172.16.1.1

```

SSH Connectivity is provided by an Azure Bastion Host. You can use the Azure Portal to logon using the Bastion
Connect functionality.

Ping to AWS from Azure
```
ping [aws_instance_ip]
$ PING 172.16.1.1 (172.16.1.1) 56(84) bytes of data.
64 bytes from 172.16.1.1: icmp_seq=1 ttl=247 time=27.5 ms
``` 

SSH requires copying your private key to the Azure instance.
```
$ ssh -i ~/.ssh/terraform-testing ec2-user@172.16.1.1
The authenticity of host '172.16.1.159 (172.16.1.159)' can't be established.
ECDSA key fingerprint is SHA256:6/cwV5RNgQzsPCJpazwlkjK2Pki7cOZ+GgSkZLvdKs0.
Are you sure you want to continue connecting (yes/no)? yes
Warning: Permanently added '172.16.1.159' (ECDSA) to the list of known hosts.

       __|  __|_  )
       _|  (     /   Amazon Linux 2 AMI
      ___|\___|___|

https://aws.amazon.com/amazon-linux-2/
```

SSH Connectivty to the AWS instance can eiher be from Azure, or adding gateway resources and public IP Resources
within AWS.

```
$ ping [azure_instance_ip]
PING 10.6.0.6 (10.6.0.6) 56(84) bytes of data.
64 bytes from 10.6.0.6: icmp_seq=1 ttl=247 time=26.3 ms
```

To SSH back into the instance use the username and password set in the Azure configuration.
```
$ ssh [azureuser]@10.6.0.6
Linux demoenv-terraform-instance 4.9.0-13-amd64 #1 SMP Debian 4.9.228-1 (2020-07-05) x86_64

The programs included with the Debian GNU/Linux system are free software;
the exact distribution terms for each program are described in the
individual files in /usr/share/doc/*/copyright.

Debian GNU/Linux comes with ABSOLUTELY NO WARRANTY, to the extent
permitted by applicable law.
```

