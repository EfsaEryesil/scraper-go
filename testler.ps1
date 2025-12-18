chcp 65001 | Out-Null
$OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8


$urls = Get-Content .\urller.txt | Where-Object { $_.Trim() -ne "" -and -not $_.Trim().StartsWith("#") }

$i = 1
foreach ($u in $urls) {
  $folder = "test{0:00}" -f $i
  New-Item -ItemType Directory -Force -Path $folder | Out-Null

  $logPath = Join-Path $folder "run.log"
  "URL: $u" | Out-File -FilePath $logPath -Encoding UTF8

  try {
    (go run . $u 2>&1) | ForEach-Object {
  $_                  
  $_ | Out-File -FilePath $logPath -Append -Encoding UTF8
}

  } catch {
    "RUN ERROR: $($_.Exception.Message)" | Out-File -FilePath $logPath -Append -Encoding UTF8
  }

  if (Test-Path .\site_data.html) { Move-Item -Force .\site_data.html (Join-Path $folder "site_data.html") }
  if (Test-Path .\screenshot.png) { Move-Item -Force .\screenshot.png (Join-Path $folder "screenshot.png") }
  if (Test-Path .\links.txt)      { Move-Item -Force .\links.txt      (Join-Path $folder "links.txt") }

  $i++
}
