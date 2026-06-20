function Get-AcceptanceMatrix {
    param(
        [string]$WorkspaceRoot = (Split-Path -Path $PSScriptRoot -Parent)
    )

    $matrixPath = Join-Path $WorkspaceRoot "scripts\local_acceptance_matrix.psd1"
    if (-not (Test-Path -LiteralPath $matrixPath)) {
        throw "Acceptance matrix not found: $matrixPath"
    }

    return Import-PowerShellDataFile -LiteralPath $matrixPath
}

function Get-AcceptanceProfileSpecs {
    param(
        [string]$WorkspaceRoot = (Split-Path -Path $PSScriptRoot -Parent),
        [string]$ProfileName
    )

    $matrix = Get-AcceptanceMatrix -WorkspaceRoot $WorkspaceRoot
    if (-not $matrix.ContainsKey($ProfileName)) {
        throw "Acceptance profile not found: $ProfileName"
    }

    return @($matrix[$ProfileName])
}

function Get-AcceptanceRoutePolicies {
    param(
        [string]$WorkspaceRoot = (Split-Path -Path $PSScriptRoot -Parent)
    )

    $matrix = Get-AcceptanceMatrix -WorkspaceRoot $WorkspaceRoot
    if (-not $matrix.ContainsKey("route_policies")) {
        throw "Acceptance route_policies not found in matrix."
    }

    return @($matrix.route_policies)
}

function Get-AcceptanceRequiredRoutes {
    param(
        [string]$WorkspaceRoot = (Split-Path -Path $PSScriptRoot -Parent)
    )

    return @(Get-AcceptanceRoutePolicies -WorkspaceRoot $WorkspaceRoot |
        Where-Object { $_.required -eq $true } |
        ForEach-Object { [string]$_.route_key })
}

function ConvertTo-JsonSafeString {
    param(
        [AllowNull()]
        [string]$Text
    )

    if ($null -eq $Text) {
        return $null
    }

    $builder = New-Object System.Text.StringBuilder
    for ($index = 0; $index -lt $Text.Length; $index++) {
        $char = $Text[$index]
        if ([char]::IsHighSurrogate($char)) {
            if ($index + 1 -lt $Text.Length -and [char]::IsLowSurrogate($Text[$index + 1])) {
                [void]$builder.Append($char)
                [void]$builder.Append($Text[$index + 1])
                $index++
                continue
            }
            [void]$builder.Append("?")
            continue
        }
        if ([char]::IsLowSurrogate($char)) {
            [void]$builder.Append("?")
            continue
        }

        $codePoint = [int][char]$char
        $isUnsafeControl = (($codePoint -lt 32) -and $char -ne "`r" -and $char -ne "`n" -and $char -ne "`t") -or $codePoint -eq 127
        if ($isUnsafeControl) {
            [void]$builder.Append("?")
            continue
        }

        [void]$builder.Append($char)
    }

    return $builder.ToString()
}

function ConvertTo-JsonSafeValue {
    param(
        $Value
    )

    if ($null -eq $Value) {
        return $null
    }

    if ($Value -is [string]) {
        return (ConvertTo-JsonSafeString -Text $Value)
    }

    if ($Value -is [System.Collections.IDictionary]) {
        $result = [ordered]@{}
        foreach ($key in $Value.Keys) {
            $result[[string]$key] = ConvertTo-JsonSafeValue -Value $Value[$key]
        }
        return $result
    }

    if ($Value -is [System.Collections.IEnumerable] -and -not ($Value -is [string])) {
        $items = @()
        foreach ($item in $Value) {
            $items += ,(ConvertTo-JsonSafeValue -Value $item)
        }
        return ,$items
    }

    $properties = @($Value.PSObject.Properties | Where-Object { $_.MemberType -eq "NoteProperty" -or $_.MemberType -eq "Property" })
    if ($properties.Count -gt 0) {
        $result = [ordered]@{}
        foreach ($property in $properties) {
            $result[$property.Name] = ConvertTo-JsonSafeValue -Value $property.Value
        }
        return $result
    }

    return $Value
}

function Write-JsonUtf8File {
    param(
        $Value,
        [string]$Path,
        [int]$Depth = 20
    )

    $safeValue = ConvertTo-JsonSafeValue -Value $Value
    $json = $safeValue | ConvertTo-Json -Depth $Depth
    [System.IO.File]::WriteAllText($Path, $json, [System.Text.UTF8Encoding]::new($false))
}
