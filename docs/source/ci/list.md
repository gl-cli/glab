---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab ci list`

Get the list of CI/CD pipelines.

```plaintext
glab ci list [flags]
```

## Examples

```console
$ glab ci list
$ glab ci list --status=failed

```

## Options

```plaintext
  -n, --name string             Return only pipelines with the given name.
  -o, --orderBy string          Order pipelines by this field. Options: id, status, ref, updated_at, user_id. (default "id")
  -F, --output string           Format output. Options: text, json. (default "text")
  -p, --page int                Page number. (default 1)
  -P, --per-page int            Number of items to list per page. (default 30)
  -r, --ref string              Return only pipelines for given ref.
      --scope string            Return only pipelines with the given scope: {running|pending|finished|branches|tags}
      --sha string              Return only pipelines with the given SHA.
      --sort string             Sort pipelines. Options: asc, desc. (default "desc")
      --source string           Return only pipelines triggered via the given source. See https://docs.gitlab.com/ci/jobs/job_rules/#ci_pipeline_source-predefined-variable for full list. Commonly used options: {merge_request_event|parent_pipeline|pipeline|push|trigger}
  -s, --status string           Get pipeline with this status. Options: running, pending, success, failed, canceled, skipped, created, manual, waiting_for_resource, preparing, scheduled
  -a, --updated-after string    Return only pipelines updated after the specified date. Expected in ISO 8601 format (2019-03-15T08:00:00Z).
  -b, --updated-before string   Return only pipelines updated before the specified date. Expected in ISO 8601 format (2019-03-15T08:00:00Z).
  -u, --username string         Return only pipelines triggered by the given username.
  -y, --yaml-errors             Return only pipelines with invalid configurations.
```

## Options inherited from parent commands

```plaintext
  -h, --help              Show help for this command.
  -R, --repo OWNER/REPO   Select another repository. Can use either OWNER/REPO or `GROUP/NAMESPACE/REPO` format. Also accepts full URL or Git URL.
```
