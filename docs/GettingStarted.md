# Getting Started
Getting Started with `terraform-provider-megaport` is easy! 

# Requirements
Before using the Megaport Terraform Provider, you will need a valid Megaport account.

To create a new account, follow the [Quick Start Guide](https://docs.megaport.com/getting-started/). Complete
the [Create an account](https://docs.megaport.com/setting-up/registering/), 
[Add company profile](https://docs.megaport.com/setting-up/registering/#adding-a-company-profile) and 
[Specify your billing market](https://docs.megaport.com/setting-up/registering/#enabling-a-billing-market) procedures.

### Passwords
If you use a Social Login (such as your Google account), you will need to set up a password on your account to use this provider. You will still be able to log in using the Social Login even when a password is set.

Remember your username and password for later use.

### Multi-Factor Authentication OTP Key
If multi-factor authentication (MFA) is enabled on your Megaport account, you need to reset MFA to get the initial key for use with the Megaport Terraform Provider:

1. Remove your current authenticator, and delete it from your MFA app (if it exists). 
1. Set up MFA again. There is an option to "enter this text code" on the "Enable Two-factor Authentication" screen within the the Megaport Portal. For details, see [Securing Your Account With Two-Factor Authentication](https://docs.megaport.com/setting-up/manage-profile/#securing-your-account-with-two-factor-authentication)
1. Note the code and enter it in the `mfa_otp_key` when configuring the provider.

