cleanup:
	rm -f *-shm *-wal *.db

rebuild: cleanup
	go build -o out/shortner ./cmd/cli/main.go
	./out/shortner seed -size 1M

