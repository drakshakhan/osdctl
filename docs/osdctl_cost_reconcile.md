## osdctl cost reconcile

Checks if there's a cost category for every OU. If an OU is missing a cost category, creates the cost category

```
osdctl cost reconcile [flags]
```

### Options

```
  -h, --help        help for reconcile
      --ou string   get OU ID
```

### Options inherited from parent commands

```
      --as string                        Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
  -a, --aws-access-key-id string         AWS Access Key ID
  -c, --aws-config string                specify AWS config file path
  -p, --aws-profile string               specify AWS profile
  -g, --aws-region string                specify AWS region (default "us-east-1")
  -x, --aws-secret-access-key string     AWS Secret Access Key
      --cluster string                   The name of the kubeconfig cluster to use
      --context string                   The name of the kubeconfig context to use
      --insecure-skip-tls-verify         If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string                Path to the kubeconfig file to use for CLI requests.
  -o, --output string                    Valid formats are ['', 'json', 'yaml', 'env']
      --request-timeout string           The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                    The address and port of the Kubernetes API server
      --skip-aws-proxy-check aws_proxy   Don't use the configured aws_proxy value
  -S, --skip-version-check               skip checking to see if this is the most recent release
```

### SEE ALSO

* [osdctl cost](osdctl_cost.md)	 - Cost Management related utilities

