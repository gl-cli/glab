<!---
Please read this!

Before opening a new issue, make sure to search for keywords in the issues
filtered by the "bug" label:

- https://gitlab.com/gitlab-org/cli/-/issues/?label_name%5B%5D=type%3A%3Abug

and verify the issue you're about to submit isn't a duplicate.
--->

### Checklist

<!-- Please test the latest versions, that will remove the possibility that you see a bug that is fixed in a newer version. -->

- [ ] I'm using the latest version of the extension (Run `glab --version`)
  - Extension version: _Put your extension version here_
- [ ] Operating system and version: _Put your version here_
- [ ] Gitlab.com or self-managed instance? _gitlab.com/self-managed instance/both_
- [ ] GitLab version (if self-managed) _GitLab version here_
  (Use the `version` endpoint, like this: gitlab.my-company.com/api/v4/version) 
- [ ] I have performed `glab auth status` to check for authentication issues

### Summary

<!-- Summarize the bug encountered concisely -->

### Environment

<!--

on POSIX system (Linux, MacOS), run

bash -c 'printf -- "- OS: %s\n- SHELL: %s\n- TERM: %s\n- GLAB: %s" "$(uname -srm)" "$SHELL" "$TERM" "$(glab --version)"'

and replace the following section with the result.

If you use non-POSIX system, fill in the section manually:

- OS: Your operating system including version and architecture (Windows 11 - AMD64, MacOS Sonoma - ARM64)
- SHELL: Your shell (bash, fish, zsh, ...)
- TERM: Your terminal emulator (Kitty, Xterm2..)
- GLAB: result of running `glab --version` (glab version 1.32.0 (2023-08-18))

-->

- OS:
- SHELL:
- TERM:
- GLAB:

<!--
Please include any other information that you believe might be relevant
in debugging. For example, you may include a shell framework like oh-my-zsh
or other customizations like editing the prompt (PS1, PS2, and others).
-->
Other:

### Steps to reproduce

<!-- How one can reproduce the issue - this is very important -->

### What is the current _bug_ behavior?

<!-- What actually happens -->

### What is the expected _correct_ behavior?

<!-- What you should see instead -->

### Relevant logs and/or screenshots

<!--- Paste the activity log from your command line -->

### Possible fixes

<!-- If you can, link to the line of code that might be responsible for the problem -->

/label ~"type::bug" ~"devops::create" ~"group::code review" ~"Category:GitLab CLI" ~"cli"
