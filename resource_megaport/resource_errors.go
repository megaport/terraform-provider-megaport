package resource_megaport

const CannotSetVIFError = "unable to set the VIF id correctly"
const CannotChangeHostedConnectionRateError = "you cannot update the rate limit on an AWSHC, create a new resource. If you create a new resource, it will need a new vLAN"
const PortNotLockedError = "the port has not been locked, modification failed"
const PortNotUnlockedError = "the port was not able to be unlocked, modification failed"
const MultipleCSPError = "multiple csp_settings are not supported"
const InvalidPartnerBEnd = "unable to find a matching partner Megaport for the specified product"
const NoMatchingPartnerPortsAtLocationError = "no matching partner ports at that location"
const TooManyPartnerPortsError = "too many results returned, please add more filter properties"
