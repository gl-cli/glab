# Add a user as a developer
$ glab repo members add --username=john.doe --role=developer
# Add a user as a maintainer with expiration date
$ glab repo members add --username=jane.smith --role=maintainer --expires-at=2024-12-31
# Add a user by ID
$ glab repo members add --user-id=123 --role=reporter
# Add a user with a custom role
$ glab repo members add --username=john.doe --role-id=101
