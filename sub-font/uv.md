```shell
uv venv --python 3.12
uv pip install fonttools brotli
uv pip freeze > ./requirements.txt
uv run pip install -r requirements.txt
```