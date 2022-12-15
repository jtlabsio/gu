BLD_DST=bin/
BLD_FLGS=-v -a -tags netgo
BNRY_NM=gu
CGO_ENABLED=0
DST=${BLD_DST}${BNRY_NM}
GO_CMD=amd64
GO_CMD=go

build: clean download
	${GO_CMD} build ${BLD_FLGS} -o ${DST} ./cmd/...

clean:
	@rm -rf ${BLD_DST}
	@mkdir ${BLD_DST}

download:
	${GO_CMD} mod download

.PHONY: build