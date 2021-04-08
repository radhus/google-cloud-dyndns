# google-cloud-dyndns

Use this tool to update a Google CloudDNS record to the address of a network interface, like a DynDNS tool.

## Usage

Download the [latest release](https://github.com/radhus/google-cloud-dyndns/releases/latest) binary for your architecture.
MIPS-hardfloat is reported to work on UniFi Security Gateway.

The tool currently only supports automatic credential fetching, i.e. by setting the `GOOGLE_APPLICATION_CREDENTIALS` environment variable. See [authentication docs](https://cloud.google.com/docs/authentication/getting-started).

Run the tool by specifying interface, host, zone and Google Cloud project:

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/etc/gcloud/credentials.json
google-cloud-dyndns -interface wan0 -host hostname.domain.tld. -zone zonename -project cloud-project
```
