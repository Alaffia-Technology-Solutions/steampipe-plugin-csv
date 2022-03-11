FROM golang:1.17.8 AS build
WORKDIR /app/
# Download go deps and cache as separate layer
COPY go.mod .
COPY go.sum .
RUN go mod download
# Build s3 plugin binary
COPY . .
RUN go build -o steampipe-plugin-s3.plugin *.go

FROM debian:buster-slim AS prod
# Install steampipe
RUN apt-get update -y && apt-get install -y curl make git
RUN curl -fsSL https://raw.githubusercontent.com/turbot/steampipe/main/install.sh | sh
# Create non-root steampipe user. This is necessary for steampipe to run
RUN adduser --system --disabled-login --ingroup 0 --gecos "steampipe user" --shell /bin/false --uid 9193 steampipe

# Set working directory to the home dir of the steampipe user
WORKDIR /home/steampipe
# Copy build artifacts from the build stage
COPY --from=build /app/steampipe-plugin-s3.plugin .
COPY --from=build /app/config/* ./.steampipe/config/
# Create steampipe config directories
RUN mkdir -p ./.steampipe/plugins/hub.steampipe.io/plugins/Alaffia-Technology-Solutions/s3@latest
# Copy s3 plugin binary to steampipe config
RUN cp steampipe-plugin-s3.plugin ./.steampipe/plugins/hub.steampipe.io/plugins/Alaffia-Technology-Solutions/s3@latest

# Grant ownership permission to the steampipe user
RUN chown -R steampipe:0 /home/steampipe

# Start steampipe
EXPOSE 9193
USER steampipe:0
# Install the aws and steampipe plugins for Steampipe (as steampipe user).
RUN steampipe plugin install steampipe aws

ENTRYPOINT ["steampipe"]
CMD ["service", "start", "--foreground", "--database-listen", "network"]