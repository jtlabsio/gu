BLD_DST=./
BLD_FLGS=-v -a -tags netgo
BNRY_NM=gu
CGO_ENABLED=0
DST=${BLD_DST}${BNRY_NM}
GO_CMD=go

build: download
	${GO_CMD} build ${BLD_FLGS} -o ${DST} ./cmd/...

download:
	${GO_CMD} mod download

.PHONY: build