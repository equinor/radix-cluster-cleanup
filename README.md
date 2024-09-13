![build workflow](https://github.com/equinor/radix-cluster-cleanup/actions/workflows/build-push.yml/badge.svg) [![SCM Compliance](https://scm-compliance-api.radix.equinor.com/repos/equinor/radix-cluster-cleanup/badge)](https://developer.equinor.com/governance/scm-policy/)

# Radix Cluster Cleanup

Keeps the Playground cluster tidy, stops and deletes "unused" applications.


go build -o rx-cleanup

LOG_LEVEL=DEBUG ./rx-cleanup --help


### Building and releasing

Deployment is managed by Flux, that monitors `master`  and `release` branches. 
Any commits here will trigger a pull requets in radix-flux to release the commit to their respective cluster

### Contributing

Want to contribute? Read our [contributing guidelines](./CONTRIBUTING.md)

---------

[Security notification](./SECURITY.md)
