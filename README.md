# OTP Brute

A high-speed, asynchronous OTP (One-Time Password) brute-force tool written in Go. Designed for security testing and pentesting.

## Installation

```bash
git clone https://github.com/devazizov/avbrute.git
cd avbrute
go build -o avbrute main.go

```

## Usage

```bash
./avbrute -u <URL> -d <DATA> [PARAMETERS]

```

**Parameters:**

- `-u` : Target URL.
- `-d` : POST request body. Replace the target code value with the word `OTP`.
- `-r` : Code length or exact range (e.g., `4` or `1000-9999`). Default: `6`.
- `-t` : Number of concurrent threads. Default: `50`.
- `-s` : Expected success HTTP status code. Default: `200`.
- `-m` : Exact text to match in the response body.
- `-c` : Cookie value to maintain session.

## Examples

**1. 4-digit code (JSON format):**

```bash
./avbrute -u [http://127.0.0.1/api/verify](http://127.0.0.1/api/verify) -d '{"phone":"998901234567","code":"OTP"}' -r 4

```

**2. 6-digit code, text match, and high threads:**

```bash
./avbrute -u [http://127.0.0.1/api/verify](http://127.0.0.1/api/verify) -d '{"phone":"998901234567","code":"OTP"}' -r 6 -m "true" -t 500

```

**3. Form-Data, specific range, and Cookie (expecting 302 Redirect):**

```bash
./avbrute -u [https://target.com/reset](https://target.com/reset) -d "csrf=token&code=OTP" -r 56000-56900 -s 302 -c "session=xyz123"

```

## Note on Performance

If the brute-force process feels slow even when using a high number of threads (`-t 1000+`), the bottleneck is the target backend server, not the tool. The maximum execution speed is entirely dependent on the server's throughput and its capacity to handle concurrent requests (e.g., a multi-worker setup). [@avdev](https://t.me/avdev)
