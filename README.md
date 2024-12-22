# Email Scraper

This project is designed to defeat as many email obfuscation methods as possible, creating a single bot capable of crawling the web and harvesting emails. It supports common and uncommon obfuscation methods such as Cloudflare email protection, ROT Cipher, HTML entity decoding, RTL (Right-to-Left) obfuscation, JavaScript-based obfuscation, SVG-encoded emails, Hex and Unicode obfuscation, object and iframe embedded addresses, JavaScript hrefs, splitting addresses with comments, Base64 encoding, basic AJAX and API request obfuscation, text-based obfuscation, and many more coming soon!

## Features

- **Email Extraction**: Scrapes email addresses from HTML content.
- **Obfuscation Handling**: Decodes obfuscated emails, including JavaScript-based methods.
- **Depth-based Crawling**: Crawls through websites up to a specified depth, staying within the domain or subdirectories.
- **Email Validation**: Validates email addresses against known standards and checks DNS records for each domain.
- **Logging**: Outputs logs to a file for debugging and analysis.

## Installation

1. Ensure Go is installed on your system. [Download Go](https://golang.org/dl/).
2. Clone the repository or download the source code.

```bash
git clone https://github.com/Pythoript/email-scraper.git
cd email-scraper
```

3. Install dependencies:

```bash
go mod tidy
```

4. Compile the project:

```bash
go build -o run
```

### Command-Line Arguments

- `URL` (required): The URL where the crawl starts.
- `-v`, `--verbose`: Enable verbose logging.
- `--disable-cookies`: Disable cookies during requests.
- `--log <logfile>`: Log output to the specified file.
- `-o`, `--output <filename>`: Output file to save scraped emails (default: `emails.txt`).
- `--skip-validation`: Skip the email validation.
- `--user-agent <user-agent>`: Custom User-Agent string for requests.
- `--max-depth <depth>`: Set the maximum crawling depth (default: 3).
- `--domain-mode <mode>`: Set crawling domain mode:
  - `1`: Stay within the current site (default).
  - `2`: Explore subdirectories.
  - `3`: Unrestricted.

### Example

To run the crawler with verbose output, skip email validation, and save emails to a file:

```bash
./run https://example.com --verbose --skip-validation --output emails.txt
```

## Functionality Breakdown

### Email Extraction

- Extracts emails from:
  - Normal email addresses found in the page content.
  - Obfuscated emails (like `data-cfemail` attributes).
  - Emails encoded in SVG images.
  - Emails obfuscated in JavaScript.

### Depth-based Crawling

The crawler supports multiple levels of recursion, allowing it to traverse deeper into a website. The `--max-depth` flag controls how many levels deep the crawler will go.

### Logging

Logs are generated for important actions, errors, and other debugging information. You can specify a log file using the `--log` flag.

## TODO

- Add OCR support.
- Capture redirects to `mailto`.
- Support CSS pseudo-element encoding.
- Remove non-visible HTML elements.

## License

This project is licensed under the AGPL-3.0 License - see the [LICENSE](LICENSE) file for details.
