sed -i 's|COPY ./cmd/scalable-ecommerce-platform ./cmd/scalable-ecommerce-platform|COPY ./cmd ./cmd|' Dockerfile
sed -i 's|COPY ./internal ./internal|COPY ./internal ./internal\nCOPY ./pkg ./pkg\nCOPY ./docs ./docs|' Dockerfile
