BUILD   := build/
TARGET  := $(BUILD)toobz
GOFLAGS := -trimpath -buildmode=pie -mod=readonly -modcacherw -buildvcs=false

all: $(TARGET)

build: $(TARGET)

$(TARGET): cmd/*.go go.mod *.go
	go build $(GOFLAGS) -o $@ cmd/main.go

check: $(TARGET)
	go test ./...

clean:
	@rm -rf $(BUILD)
