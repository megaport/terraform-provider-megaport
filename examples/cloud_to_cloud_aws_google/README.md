This examples provisions a full multi-cloud demonstration environment including networking and compute instances. It 
requires account credentials for Megaport, Amazon Web Services, and Google Cloud Platform.

This example requires some prior understanding of AWS and GCP platforms, as well as usage of SSH and key pairs.  

## Before you begin
  * Complete the [Getting Started Requirements](https://registry.terraform.io/providers/megaport/megaport/latest/docs/guides/gettingstarted)
  * You will also need to create a `gcp-credentials.json` file in this directory with the 
    service account credentials for this service account: `615045253644-compute@developer.gserviceaccount.com`. 
    For more information on creating the account keys, see 
    [GCP Managing Service Account Keys](https://cloud.google.com/iam/docs/creating-managing-service-account-keys)  
  * You will also need valid AWS credentials in your OS shell. The 
    [AWS CLI documentation](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html) details 
    the methods for configuring AWS Credentials.  
  * Create a key-pair in AWS for the compute instance and update the `aws_ec2_key_pair_name` in `variables.tf`,
    the default key-pair name is `terraform-testing`

## Notes
You cannot run this at the same time as another person, *unless* you change the
resource names in the GCP configuration file. This is because GCP uses the names
of resources as unique identifiers, and it will attempt to create resources that
already "exist".  

This example will not work on the Megaport Staging environment. This is because it requires
real connections to AWS and GCP.

When you have completed, use `terraform destroy` so that you will stop incurring costs for the resources.

## Testing

Once Terraform has completed the build it will output the IP addresses of the compute resources, similar to:
```
aws_instance_ip = 172.16.1.1
gcp_instance_ip = 192.168.0.1
ssh_command = ssh -i ~/.ssh/terraform-testing        ec2-user@172.16.1.1

```

SSH Connectivity to the GCP instance is easy via the GCP console.

Ping to AWS from GCP
```
ping [aws_instance_ip]
$ PING 172.16.1.1 (172.16.1.1) 56(84) bytes of data.
64 bytes from 172.16.1.1: icmp_seq=1 ttl=247 time=27.5 ms
``` 

SSH requires copying your private key to the GCP instance.
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

SSH Connectivty to the AWS instance can eiher be from GCP, or adding gateway resources and public IP Resources
within AWS.

```
$ ping [gcp_instance_ip]
PING 192.168.1.1 (192.168.1.1) 56(84) bytes of data.
64 bytes from 192.168.1.1: icmp_seq=1 ttl=247 time=26.3 ms
```

SSH back to the GCP instance requires setting up a public key on the GCP instance with a private key on the
AWS instance.
```
$ ssh -i [private-key] [your-login]@192.168.1.1
Linux demoenv-terraform-instance 4.9.0-13-amd64 #1 SMP Debian 4.9.228-1 (2020-07-05) x86_64

The programs included with the Debian GNU/Linux system are free software;
the exact distribution terms for each program are described in the
individual files in /usr/share/doc/*/copyright.

Debian GNU/Linux comes with ABSOLUTELY NO WARRANTY, to the extent
permitted by applicable law.
```

