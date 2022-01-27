This will provision an MCR (Megaport Cloud Router) connected to AWS over a Hosted Connection with BGP and BFD configuration.

Before you begin, you need to have completed the [Getting Started Requirements](https://registry.terraform.io/providers/megaport/megaport/latest/docs/guides/gettingstarted)  

Replace the `username`, `password` and optional `mfa_otp_key` with your own credentials.

This configuration will deploy on the staging environment. To use this on production, valid CSP attributes are required:
+ `megaport_aws_connection.amazon_account`
