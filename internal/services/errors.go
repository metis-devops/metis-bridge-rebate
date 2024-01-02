package services

type ErrorNoNeedToTransfer struct {
	msg string
}

func (e ErrorNoNeedToTransfer) Error() string {
	return e.msg
}
