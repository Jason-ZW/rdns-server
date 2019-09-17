package consts

const (
	DBDriverName         = "mysql"
	DBMaxOpenConnections = 2000
	DBMaxIdleConnections = 1000

	EnvExpire             = "EXPIRE"
	EnvRotate             = "ROTATE"
	ExpireIntervalSeconds = 600
)
