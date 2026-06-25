```shell
uv venv --python 3.13
uv pip install fonttools brotli
uv pip freeze > ./requirements.txt
uv run pip install -r requirements.txt

uv run python ./subset.py
uv run python ./run_local.py
```

## 传入环境变量运行

**PowerShell:**
```powershell
$env:GH_TOKEN="your_token_here"; uv run python ./run_local.py
```

**Bash:**
```bash
GH_TOKEN="your_token_here" uv run python ./run_local.py
```

可同时指定代理：

**PowerShell:**
```powershell
$env:GH_TOKEN="your_token_here"; $env:HTTP_PROXY="http://127.0.0.1:7890"; uv run python ./run_local.py
```

**Bash:**
```bash
GH_TOKEN="your_token_here" HTTP_PROXY="http://127.0.0.1:7890" uv run python ./run_local.py
```