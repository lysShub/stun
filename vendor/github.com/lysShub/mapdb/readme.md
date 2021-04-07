# mapdb 

使用map数据结构实现的简单缓存数据结构，支持表结构数据(二维数据)。并且支持TTL；特别适用于IP白名单等场景下使用。



### 开始

GO111MODULE=on

[参考](https://github.com/lysShub/mapdb/blob/master/test/test.go)



### 性能



```cmd
# Comprehensive 综合改写查，有11次操作
goos: windows
goarch: amd64
pkg: github.com/lysShub/mapdb/test/test_prop
BenchmarkComprehensive-8   	 2675017	       451 ns/op	     151 B/op	       3 allocs/op
PASS
ok  	github.com/lysShub/mapdb/test/test_prop	1.881s
```

