# Installation
There are three methods of installing the provider: 

* [From the Terraform Provider Registry](https://registry.terraform.io/providers/megaport/megaport/latest) (preferred)
* [Pre-built releases](#installing-a-pre-built-release) (suitable for most use cases)
* [Install from source](#installing-from-source)

## Install from the Terraform Provider Registry
Include the provider block in your code and run `terraform init`
```
terraform {
  required_providers {
    megaport = {
      source = "megaport/megaport"
      version = "0.1.1"
    }
  }
}
```

## Installing a Pre-Built Release

### Terraform v0.13 or higher  
Create the `.terraform.d/` directory in your home folder.

```
cd ~
ARCH=$(go version | cut -d" " -f4 | sed 's/\//_/g')
PLUGIN_DIRECTORY="$(pwd)/.terraform.d/plugins/megaport/megaport/0.1.0/${ARCH}/"
mkdir -p $PLUGIN_DIRECTORY
echo $PLUGIN_DIRECTORY
```

Download the relevant binary for your system from the [releases](https://github.com/megaport/terraform-provider-megaport/releases). Once the download is complete, 
rename it to `terraform-provider-megaport`. Copy the executable into the location specified by `$PLUGIN_DIRECTORY`.

In your Terraform code, include a mapping for the locally hosted provider: 
```
terraform {
  required_providers {
    megaport = {
      source  = "megaport.com/megaport/megaport"
      version = "0.1.0"
    }
  }
}
```

### Terraform Version 0.12 or Lower
Create the `.terraform.d/` directory in your home folder.

```
cd ~
ARCH=$(go version | cut -d" " -f4 | sed 's/\//_/g')
PLUGIN_DIRECTORY="$(pwd)/.terraform.d/plugins/${ARCH}/"
mkdir -p $PLUGIN_DIRECTORY
echo $PLUGIN_DIRECTORY
```

Download the relevant binary for your system from the [releases](https://github.com/megaport/terraform-provider-megaport/releases). Once the download is complete, 
rename it to `terraform-provider-megaport`. Copy the executable into the location specified by `$PLUGIN_DIRECTORY`.

## Installing from Source

1. Ensure that Go is installed (at least `v1.13`). You can do this by running the command: 
```
$ go version
go version go1.13.8 linux/amd64
```

1. Clone the repository to your local machine. 
1. Run `./build.sh`, which will build the provider and return the location of the executable file.
```
$ ./build.sh 
Provider built at 'bin/terraform-provider-megaport_beta0.1-11-g3987118'.
Symbolic link created from build directory to terraform.d. < 0.13
Symbolic link created from build directory to terraform.d. >= 0.13
```
That's it!

# Using the Provider

To use the provider, initialize it and ensure that it is up and running. Then, ensure that Terraform is picking it up correctly. 

Perform these steps in the directory of your Terraform project:

1. Run `terraform init` and verify that the provider was installed with `terraform version`.
```
$ terraform init

Initializing the backend...

Initializing provider plugins...

Terraform has been successfully initialized!

You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.
```

If you set or change modules or backend configuration for Terraform, rerun this command to reinitialize your working directory. If you forget, other commands will detect it and remind you.
```
$ terraform version
Terraform v0.13.0
+ provider.megaport (unversioned)
```

