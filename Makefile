cleanup:
	rm -f *-shm *-wal *.db

rebuild: cleanup
	go build -o out/shortner ./cmd/cli/main.go
	./out/shortner seed -size 1M

containerise:
	podman build -t golang-server-with-memcached .
	podman run -p 9091:9091 -p 11211:11211 golang-server-with-memcached
