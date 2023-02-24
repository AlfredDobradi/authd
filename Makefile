user := alfreddobradi
app := authd

docker:
	docker buildx build --push \
		-o type=image \
		--platform=linux/arm64 \
		--platform=linux/amd64 \
		--tag ${user}/${app}:latest \
		--tag ${user}/${app}:${AUTHD_VERSION} \
		--tag registry.0x42.in/${user}/${app}:latest \
		--tag registry.0x42.in/${user}/${app}:${AUTHD_VERSION} .