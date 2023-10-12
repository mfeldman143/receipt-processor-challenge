# Use the official Golang image as the base image
FROM golang:1.17-alpine

# Set the working directory in the container
WORKDIR /app

# Copy Go mod and sum files to download dependencies
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy the Go source code to the container
COPY . .

# Build the Go application
RUN go build -o main .

# Expose the application port
EXPOSE 8080

# Start the application
CMD [ "./main" ]
