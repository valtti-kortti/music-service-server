package config

type Config struct {
	Youtube Youtube
	Rest    Rest
}

type Youtube struct {
	Token string `envconfig:"TOKEN"`
	Limit int64  `envconfig:"LIMIT"`
}

type Rest struct {
	Address           string `envconfig:"ADDRESS"`
	ReadTimeout       int64  `envconfig:"READ_TIMEOUT"`
	WriteTimeout      int64  `envconfig:"WRITE_TIMEOUT"`
	ReadHeaderTimeout int64  `envconfig:"READ_HEADER_TIMEOUT"`
	IdleTimeout       int64  `envconfig:"IDLE_TIMEOUT"`
}
