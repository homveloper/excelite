[CmdletBinding()]
param (
    [Parameter(Mandatory=$false)]
    [switch]$help,

    [Parameter(Position=0, Mandatory=$false,
               HelpMessage="Directory path to search files in")]
    [string]$path = ".",

    [Parameter(Position=1, Mandatory=$false,
               HelpMessage="File patterns to search for (comma-separated, e.g., '*.cpp,*.h')")]
    [string]$filePattern = "*.go",
    
    [Parameter(Position=2, Mandatory=$false,
               HelpMessage="Prefix for output files")]
    [string]$outputPrefix = "output_",
    
    [Parameter(Position=3, Mandatory=$false,
               HelpMessage="Number of files to split into (must be > 0)")]
    [int]$splitCount = 1
)

# Check for help first, before any other parameter validation
if ($help -or $args -contains "--help" -or $args -contains "-h" -or $args -contains "/?") {
    $scriptName = $MyInvocation.ScriptName
    @"
USAGE:
    $scriptName [-path <directory>] [-filePattern <patterns>] [-outputPrefix <prefix>] [-splitCount <number>]

PARAMETERS:
    -path        : Directory to search files in (e.g., "C:\Project" or "..\src")
                  Default: current directory
    -filePattern : Comma-separated file patterns (e.g., "*.cpp,*.h" or "*.go,*.mod")
                  Default: "*.go"
    -outputPrefix: Prefix for output files (will create prefix1.txt, prefix2.txt, etc.)
                  Default: "output_"
    -splitCount  : Number of files to split into (must be greater than 0)
                  Default: 5

EXAMPLES:
    $scriptName -help
    $scriptName -path "C:\Project" -filePattern "*.cpp,*.h" -outputPrefix "cpp_files_" -splitCount 5
    $scriptName -path "..\src" -filePattern "*.go,*.mod"
    $scriptName                     # Uses all default values
"@
    exit 0
}

# Input validation
$inputErrors = @()

if ([string]::IsNullOrWhiteSpace($path)) {
    $inputErrors += "Path cannot be empty"
} elseif (-not (Test-Path $path)) {
    $inputErrors += "Directory not found: $path"
}

if ([string]::IsNullOrWhiteSpace($filePattern)) {
    $inputErrors += "File pattern cannot be empty"
}

if ([string]::IsNullOrWhiteSpace($outputPrefix)) {
    $inputErrors += "Output prefix cannot be empty"
}

if ($splitCount -lt 1) {
    $inputErrors += "Split count must be greater than 0"
}

if ($inputErrors.Count -gt 0) {
    Write-Host "ERROR: The following problems were found:" -ForegroundColor Red
    $inputErrors | ForEach-Object { Write-Host "- $_" -ForegroundColor Red }
    Write-Host ""
    Show-Usage
    exit 1
}

# Convert to absolute path
$absolutePath = Resolve-Path $path
Write-Host "Searching in: $absolutePath"

# Split the file patterns and get all matching files
$patterns = $filePattern.Split(',').Trim()
$files = @()
foreach ($pattern in $patterns) {
    $matchedFiles = Get-ChildItem -Path $absolutePath -Recurse -Filter $pattern
    $files += $matchedFiles
    Write-Host "Found $($matchedFiles.Count) files matching pattern '$pattern'"
}

$totalFiles = $files.Count

if ($totalFiles -eq 0) {
    Write-Host "ERROR: No files found matching patterns: $filePattern in directory: $absolutePath" -ForegroundColor Red
    Write-Host ""
    exit 1
}

# Calculate files per output file
$filesPerGroup = [math]::Ceiling($totalFiles / $splitCount)

Write-Host "Found total $totalFiles files matching patterns '$filePattern'"
Write-Host "Splitting into $splitCount files with approximately $filesPerGroup files each"

# Create output files in the current directory
$currentDir = Get-Location
for ($i = 0; $i -lt $splitCount; $i++) {
    $outputFile = Join-Path $currentDir "$outputPrefix$($i+1).txt"
    
    # Get the files for this group
    $start = $i * $filesPerGroup
    $groupFiles = $files | Select-Object -Skip $start -First $filesPerGroup
    
    # Clear the file if it exists
    if (Test-Path $outputFile) {
        Clear-Content $outputFile
    }
    
    # Process each file in the group
    $groupCount = 0
    foreach ($file in $groupFiles) {
        Add-Content $outputFile "========= $($file.FullName) ========="
        Get-Content $file.FullName | Add-Content $outputFile
        Add-Content $outputFile "`n"
        $groupCount++
    }
    
    Write-Host "Created $outputFile with $groupCount files"
}

Write-Host "`nSplit complete! Files are saved in current directory: $currentDir"