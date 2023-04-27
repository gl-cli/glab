---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

# Add a new configuration

To add a new configuration to `config.yaml.lock`:

1. Add a head comment and a default value:

   ```yaml
   # head comment
   new_key: default_value
   ```

1. Add any configuration that is specific to a hostname or GitLab instance to the
   `hosts` section, in the `gitlab.com` subsection:

   ```yaml
   ...
   # This configuration is specifically for GitLab instances
   hosts:
     gitlab.com:
       ...
       # This is a new config
       new_key: default_value
   ...
   ```

1. Add general configuration changes before the `hosts` section:

   ```yaml
   ...
   # Head comment
   new_key: default_value
   # Configuration specific for GitLab instances
   hosts:
     gitlab.com:
   ...
   ```

1. Run `make gen-config` or `cd internal/config && go generate`.
1. Most configuration keys can be overwritten by their corresponding environment variables.
   If the corresponding environment variable name differs from the configuration key's name,
   set the environment variable's name in the `config_mapping.go` file.

