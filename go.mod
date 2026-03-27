module github.com/abdelaaziz0/kerbrutal

go 1.13

require (
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/ropnop/gokrb5/v8 v8.0.0-20201111231119-729746023c02
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1 // indirect
	golang.org/x/net v0.50.0
	golang.org/x/text v0.35.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

replace github.com/ropnop/gokrb5/v8 => ./gokrb5
