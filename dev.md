```shell
cd VincentZyuApps.github
# cd D:\aaaStuffsaaa\from_git\github\VincentZyuApps.github
cd sub-font
uv venv --python 3.12
# uv pip install fonttools brotli
# uv pip freeze > ./requirements.txt
uv run pip install -r requirements.txt
uv run python .\subset.py

$env:GH_TOKEN = "ghp_你的token"

cd ..
# cd D:\aaaStuffsaaa\from_git\github\VincentZyuApps.github
cd line-stats
go run main.go --proxy http://192.168.31.233:7890 --tmp

cd ..
# cd D:\aaaStuffsaaa\from_git\github\VincentZyuApps.github
cd lang-stats
go run main.go --proxy http://192.168.31.233:7890 --tmp

cd ..
# cd D:\aaaStuffsaaa\from_git\github\VincentZyuApps.github
cd git-stats
go run main.go --proxy http://192.168.31.233:7890 --tmp


```