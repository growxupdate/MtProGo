.PHONY: tidy test vet ci verify-clean

tidy:
	go mod tidy

test:
	go test ./...

vet:
	go vet ./...

ci: tidy test vet verify-clean

verify-clean:
	@test "$$(go list -m all | wc -l)" = "1" || (echo "unexpected external modules" && go list -m all && exit 1)
	@test ! -s go.sum || (echo "go.sum should be empty or absent for V8" && exit 1)
