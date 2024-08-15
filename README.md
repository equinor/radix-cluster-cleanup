![build workflow](https://github.com/equinor/radix-cluster-cleanup/actions/workflows/build-push.yml/badge.svg) [![SCM Compliance](https://scm-compliance-api.radix.equinor.com/repos/equinor/8de2870b-4681-4c54-8f5e-2cb7a85f3201/badge)](https://developer.equinor.com/governance/scm-policy/)


go build -o rx-cleanup

LOG_LEVEL=DEBUG ./rx-cleanup --help


### Building and releasing

We are making releases available as GitHub releases using [go-releaser](https://goreleaser.com/). The release process is controlled by the `.goreleaser.yml` file.

To make a release:
1. Set the version number in the constant `version` in the file `cmd/version.go`
2. Ensure there is no `dist` folder in the project (left from previous release)
3. Get the [Personal Access Token](https://github.com/settings/tokens) (PAT) - with access to repository and `write:packages` scope, and with enabled SSO for organisation (or create it)
4. Login to the docker repository with your PAT
    ```
    echo $CR_PAT | docker login ghcr.io -u magnus-longva-bouvet --password-stdin
    ```
5. Run the command to create a version with a tag, build a docker image and push them to GitHub repository. Recommended to set <number_of_cores> to a number well below your CPU count if you want to get any work done while compiling.
    ```
    git tag -a v<version_numnber> -m "<release_note>"
	git push origin v<version_number>
    goreleaser release --rm-dist --parallelism <number_of_cores>
    ```
6. If something goes wrong:
    - open the GitHub repository and delete [created tag](https://github.com/equinor/radix-cluster-cleanup/releases/) (with release)
    - delete it locally ` git tag -d v0.0.1`
    - reset changes `git reset --hard`
    - delete the `dist` folder
    - perform the previous step `make release ...` again

To generate a local version for debugging purposes, it can be built using:

```
CGO_ENABLED=0 GOOS=darwin go build -ldflags "-s -w" -a -installsuffix cgo -o ./rx-cleanup
```

### Contributing

Want to contribute? Read our [contributing guidelines](./CONTRIBUTING.md)

---------

[Security notification](./SECURITY.md)
