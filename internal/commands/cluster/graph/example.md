# Run the default query for agent 123
$ glab cluster graph -R user/project -a 123

# Show common resources from the core and RBAC groups
$ glab cluster graph -R user/project -a 123 --core --rbac

# Show certain resources
$ glab cluster graph -R user/project -a 123 --resources=pods --resources=configmaps

# Same as above, but more compact
$ glab cluster graph -R user/project -a 123 -r={pods,configmaps}

# Select a certain namespace
$ glab cluster graph -R user/project -a 123 -n={my-ns,my-stuff}

# Select all namespaces that have a certain annotation
$ glab cluster graph -R user/project -a 123 --ns-expression='"my-annotation" in annotations'

# Advanced usage - pass the full query directly via stdin.
# The query below watches service accounts in all namespaces except for the kube-system.
$ Q='{"queries":[{"include":{"resource_selector_expression":"resource == \"serviceaccounts\""}}],"namespaces":{"object_selector_expression":"name != \"kube-system\""}}'

$ echo -n "$Q" | glab cluster graph -R user/project -a 123 --stdin
