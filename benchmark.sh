#!/bin/bash

# Build papyrus
go build -o papyrus cmd/papyrus/main.go

# Create a clean HTML version of the invoice for Chrome
sed -e 's/<page.*//' -e 's/<\/\?header>//g' -e 's/<\/\?footer>//g' -e 's/<page-number\/>//g' -e 's/<page-count\/>//g' examples/invoice.xml > invoice.html

# Find Chrome executable
if command -v google-chrome &> /dev/null; then
    CHROME="google-chrome"
elif command -v google-chrome-stable &> /dev/null; then
    CHROME="google-chrome-stable"
elif command -v chromium &> /dev/null; then
    CHROME="chromium"
elif command -v chromium-browser &> /dev/null; then
    CHROME="chromium-browser"
else
    echo "Chrome/Chromium not found!"
    exit 1
fi

echo "====================================="
echo "        Real-world Benchmark         "
echo "====================================="
echo ""

echo "1. papyrus (Pure Go engine)"
echo "-------------------------------------"
# We run it once to warm up the OS font/disk cache just to be perfectly fair to Chrome
./papyrus examples/invoice.xml test1.pdf > /dev/null 2>&1
# Measure
time ./papyrus examples/invoice.xml invoice_papyrus.pdf
echo ""

echo "2. Headless Chrome"
echo "-------------------------------------"
# Warmup Chrome (Chrome has a huge cold start, so warming it up is fair)
$CHROME --headless --disable-gpu --no-sandbox --print-to-pdf=test2.pdf invoice.html > /dev/null 2>&1
# Measure
time $CHROME --headless --disable-gpu --no-sandbox --print-to-pdf=invoice_chrome.pdf invoice.html > /dev/null 2>&1
echo ""

echo "====================================="
echo "          File Size Output           "
echo "====================================="
ls -lh invoice_papyrus.pdf invoice_chrome.pdf
