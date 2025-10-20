# clean_and_reload.ps1

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Nettoyage et rechargement des donnees" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Étape 1 : Supprimer les bases sur TOUS les nœuds
Write-Host "Etape 1/4 : Suppression des bases sur tous les noeuds..." -ForegroundColor Yellow

$nodes = @("5987", "5988", "5989", "5990")
$databases = @("ecommerce_orders", "ecommerce_products", "ecommerce_sellers", "ecommerce_leads")

foreach ($node in $nodes) {
    Write-Host "  Noeud localhost:$node..." -ForegroundColor Gray
    foreach ($db in $databases) {
        $result = curl.exe -s -X DELETE "http://admin:ecommerce2024@localhost:$node/$db" 2>$null
        Write-Host "    $db supprime" -ForegroundColor Gray
    }
}

# Étape 2 : Attendre la synchronisation
Write-Host ""
Write-Host "Etape 2/4 : Attente synchronisation (10 secondes)..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# Étape 3 : Recréer les bases sur le nœud principal
Write-Host ""
Write-Host "Etape 3/4 : Recreation des bases sur NA1..." -ForegroundColor Yellow

foreach ($db in $databases) {
    $result = curl.exe -s -X PUT "http://admin:ecommerce2024@localhost:5987/$db"
    Write-Host "  $db cree" -ForegroundColor Green
}

Write-Host ""
Write-Host "Etape 4/4 : Chargement des donnees..." -ForegroundColor Yellow
Write-Host ""

# Étape 4 : Charger les données
.\bin\loader.exe -csv .\data -verbose

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "Chargement termine!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green