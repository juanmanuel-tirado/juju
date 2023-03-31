# discourse-sync

This Python script is used to sync our CLI docs to Discourse using the output
of the `juju documentation` command. It requires the following environment
variables to be set:

| Variable name            | Description                                                                                                                                                                |
|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `DISCOURSE_HOST`         | URL for the Discourse server to sync docs to. If not set, this defaults to `https://discourse.charmhub.io/`.                                                               |
| `DISCOURSE_API_USERNAME` | Username to use for Discourse API requests. Must be a user with access to the API key provided in `DISCOURSE_API_KEY`. Use your own Discourse username if running locally. |
| `DISCOURSE_API_KEY`      | [API key](https://meta.discourse.org/t/create-and-configure-an-api-key/230124) for accessing the Discourse server's API.                                                   |
| `DOCS_DIR`               | Path to a directory containing Markdown files to sync (i.e. the argument provided to the `--out` flag of `juju documentation`).                                            |
| `POST_IDS`               | Path to a YAML file mapping each doc name to its post ID on Discourse.                                                                                                     |

This script runs in two different modes, which have different behaviour when a
doc is discovered with no corresponding entry in the `POST_IDS` file:
- When running locally, we create a new Discourse post for the doc, and add the
  URL as a new entry in the `POST_IDS` file.
- When running in CI, we can't make persistent changes to the `POST_IDS` file,
  so just log a warning.