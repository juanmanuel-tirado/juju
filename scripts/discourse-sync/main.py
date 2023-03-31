import os
import yaml
from pydiscourse import DiscourseClient


def main():
    # Get configuration from environment variables
    DISCOURSE_HOST = os.environ.get('DISCOURSE_HOST', 'https://discourse.charmhub.io/')
    DISCOURSE_API_USERNAME = os.environ.get('DISCOURSE_API_USERNAME')
    DISCOURSE_API_KEY = os.environ.get('DISCOURSE_API_KEY')
    DOCS_DIR = os.environ.get('DOCS_DIR')
    POST_IDS = os.environ.get('POST_IDS')

    client = DiscourseClient(
        host=DISCOURSE_HOST,
        api_username=DISCOURSE_API_USERNAME,
        api_key=DISCOURSE_API_KEY,
    )

    with open(POST_IDS, 'r') as file:
        post_ids = yaml.safe_load(file)

    for entry in os.scandir(DOCS_DIR):
        if not is_markdown_file(entry):
            print(f'skipping file {entry.name}: not a Markdown file')
            continue

        doc_name = removesuffix(entry.name, ".md")
        content = open(entry.path, 'r').read()

        # print(post_ids)
        if post_ids and doc_name in post_ids:
            # Update Discourse post
            print(f'updating doc {doc_name} with post_id {post_ids[doc_name]}')
            client.update_post(
                post_id=post_ids[doc_name],
                content=content,
            )

        else:
            # Create new Discourse post
            print(f'no post_id found for doc {doc_name}: creating new post')
            post = client.create_post(
                title=f'juju {doc_name}',
                category_id=22,
                content=content,
                tags=['olm'],
            )
            # Save post ID in yaml map for later
            post_ids[doc_name] = post['id']
            print(f'created post #{post["id"]} for doc {doc_name}')

            with open(POST_IDS, 'w') as file:
                yaml.safe_dump(post_ids, file)


def is_markdown_file(entry: os.DirEntry) -> bool:
    return entry.is_file() and entry.name.endswith(".md")


def removesuffix(text, suffix):
    if suffix and text.endswith(suffix):
        return text[:-len(suffix)]
    return text


if __name__ == "__main__":
    main()
