module yelo/go-util

go 1.12

replace (
	golang.org/x/crypto => github.com/golang/crypto v0.0.0-20200128174031-69ecbb4d6d5d
	golang.org/x/sys => github.com/golang/sys v0.0.0-20200202164722-d101bd2416d5
)

require (
	github.com/emirpasic/gods v1.12.0
	github.com/go-redis/redis v6.15.7+incompatible
	github.com/json-iterator/go v1.1.9
)
