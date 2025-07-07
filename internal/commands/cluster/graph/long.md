This command starts a web server that shows a live view of the Kubernetes object graph in a browser.
It uses the GitLab Agent for Kubernetes running in the cluster.
It requires:

- Version 18.1 or later of GitLab and the GitLab Agent.
- At least the Developer role in the agent project.
- This command requires a personal access token or project access token
  for authentication. The token must have the `read_api` and `k8s_proxy` scopes.

Leave feedback in [issue 7900](https://gitlab.com/gitlab-org/cli/-/issues/7900).

### Resource filtering

To filter resources and namespaces, use [CEL expressions](https://cel.dev/).

`object_selector_expression`: Filters objects. The expression must return a boolean. These variables are available:

- `obj`: The Kubernetes object being evaluated.
- `group`: The group of the object.
- `version`: The version of the object.
- `resource`: The resource name of the object, like `pods` for the `Pod` kind.
- `namespace`: The namespace of the object.
- `name`: The name of the object.
- `labels`: The labels of the object.
- `annotations`: The annotations of the object.

`resource_selector_expression`: Filters Kubernetes discovery information to include or exclude resources
from the watch request. The expression must return a boolean. These variables are available:

- `group`: The group of the object.
- `version`: The version of the object.
- `resource`: The resource name of the object, like `pods` for the `Pod` kind.
- `namespaced`: The scope of group, version, and resource. Can be `bool`, `true`, or `false`.

For more information about using [label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors)
and [field selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/) to select namespaces, see the Kubernetes documentation.

### Advanced usage

Apart from high-level ways to construct the query, this command enables
you to construct and send the query using all underlying API features.
To understand what is possible, and how to do it, see the
[technical design doc](https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/blob/master/doc/graph_api.md)

The user should have permission to access the agent project.
For more information, see [Grant users Kubernetes access](https://docs.gitlab.com/user/clusters/agent/user_access/).
