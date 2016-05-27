# gitbook-watcher

## 简介

gitbook-watcher 是一个用 golang 实现的 gitbook 本地辅助工具。项目地址：[https://github.com/hiant/gitbook-watcher](https://github.com/hiant/gitbook-watcher)

gitbook本身自带的livereload，使用nodejs做的监控。对于某些内存比较小的机器来说，本地开发一段时间，多次保存触发livereload之后，node应用的

## 安装编译

1. 安装并部署go
2. 使用 `git clone https://github.com/hiant/gitbook-watcher.git` 检出项目或者在项目主页下载 [zip压缩包](https://codeload.github.com/hiant/gitbook-watcher/zip/master) 解压后，进到目录执行 go build watcher.go 进行编译
3. 项目中包含在win7x64环境下编译的可执行程序，应该可以直接使用

## 使用

将可执行程序放到你的book下，和book.json平级。
  ```
  Usage of watcher:
    -path string
          Watcher path (default ".")
    -port string
          Listening port (default "4000")
  ```
