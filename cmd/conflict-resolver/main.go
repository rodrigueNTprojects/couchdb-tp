// cmd/conflict-resolver/main.go
// Programme principal pour résoudre les conflits de réplication
package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    
    "couchdb-tp/pkg/client"
    "couchdb-tp/pkg/conflict"
)

func main() {
    var (
        database = flag.String("database", "", "Nom de la base de données")
        docID    = flag.String("doc", "", "ID du document à résoudre")
        listOnly = flag.Bool("list", false, "Lister seulement les conflits sans les résoudre")
        nodeURL  = flag.String("node", "http://localhost:5987", "URL du nœud CouchDB")
        username = flag.String("user", "admin", "Nom d'utilisateur")
        password = flag.String("pass", "ecommerce2024", "Mot de passe")
        verbose  = flag.Bool("verbose", false, "Mode verbose")
    )
    flag.Parse()
    
    // Configuration des logs
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    
    log.Println("Résolveur de Conflits CouchDB")
    log.Println("============================")
    
    // Validation des paramètres
    if *database == "" {
        fmt.Println("Usage: conflict-resolver -database <db_name> [-doc <doc_id>] [options]")
        fmt.Println("")
        fmt.Println("Options:")
        fmt.Println("  -database string    Nom de la base de données (requis)")
        fmt.Println("  -doc string         ID du document à résoudre")
        fmt.Println("  -list               Lister seulement les conflits")
        fmt.Println("  -node string        URL du nœud CouchDB (défaut: http://localhost:5987)")
        fmt.Println("  -user string        Nom d'utilisateur (défaut: admin)")
        fmt.Println("  -pass string        Mot de passe (défaut: ecommerce2024)")
        fmt.Println("  -verbose            Mode verbose")
        fmt.Println("")
        fmt.Println("Exemples:")
        fmt.Println("  # Lister les conflits dans ecommerce_orders")
        fmt.Println("  ./conflict-resolver -database ecommerce_orders -list")
        fmt.Println("")
        fmt.Println("  # Résoudre un conflit spécifique")
        fmt.Println("  ./conflict-resolver -database ecommerce_orders -doc test_order_conflict")
        fmt.Println("")
        fmt.Println("  # Résoudre sur un nœud spécifique")
        fmt.Println("  ./conflict-resolver -database ecommerce_leads -doc lead_sync_conflict -node http://localhost:5988")
        os.Exit(1)
    }
    
    if *verbose {
        log.Printf("Connexion à %s", *nodeURL)
        log.Printf("Base de données: %s", *database)
        if *docID != "" {
            log.Printf("Document: %s", *docID)
        }
    }
    
    // Création du client CouchDB
    couchClient, err := client.New(*nodeURL, *username, *password)
    if err != nil {
        log.Fatalf("Erreur création client CouchDB: %v", err)
    }
    
    // Création du résolveur de conflits
    resolver := conflict.NewConflictResolver(couchClient, *database)
    
    if *listOnly {
        // Mode listing des conflits
        log.Printf("Recherche des conflits dans la base '%s'...", *database)
        
        conflicts, err := resolver.ListConflicts()
        if err != nil {
            log.Fatalf("Erreur listing conflits: %v", err)
        }
        
        if len(conflicts) == 0 {
            log.Println("Aucun conflit détecté dans cette base de données")
        } else {
            log.Printf("Conflits détectés (%d documents):", len(conflicts))
            for i, conflictDoc := range conflicts {
                log.Printf("  %d. %s", i+1, conflictDoc)
            }
            
            log.Println("")
            log.Println("Pour résoudre un conflit spécifique:")
            log.Printf("  ./conflict-resolver -database %s -doc <document_id>", *database)
        }
        
    } else if *docID != "" {
        // Mode résolution d'un document spécifique
        log.Printf("Résolution du conflit pour le document '%s'...", *docID)
        
        err := resolver.ResolveByTimestamp(*docID)
        if err != nil {
            log.Fatalf("Erreur résolution conflit: %v", err)
        }
        
        log.Printf("Résolution terminée pour le document '%s'", *docID)
        
        if *verbose {
            log.Println("Vérification post-résolution...")
            conflicts, err := resolver.ListConflicts()
            if err == nil {
                remaining := 0
                for _, doc := range conflicts {
                    if doc == *docID {
                        remaining++
                    }
                }
                if remaining == 0 {
                    log.Println("✓ Conflit résolu avec succès")
                } else {
                    log.Println("⚠ Des conflits persistent")
                }
            }
        }
        
    } else {
        // Mode résolution automatique de tous les conflits
        log.Printf("Recherche et résolution automatique dans '%s'...", *database)
        
        conflicts, err := resolver.ListConflicts()
        if err != nil {
            log.Fatalf("Erreur listing conflits: %v", err)
        }
        
        if len(conflicts) == 0 {
            log.Println("Aucun conflit à résoudre")
            return
        }
        
        log.Printf("Résolution de %d conflits...", len(conflicts))
        
        resolved := 0
        errors := 0
        
        for _, conflictDoc := range conflicts {
            log.Printf("Résolution: %s", conflictDoc)
            
            err := resolver.ResolveByTimestamp(conflictDoc)
            if err != nil {
                log.Printf("Erreur résolution %s: %v", conflictDoc, err)
                errors++
            } else {
                resolved++
                if *verbose {
                    log.Printf("✓ %s résolu", conflictDoc)
                }
            }
        }
        
        log.Printf("Résolution terminée: %d résolus, %d erreurs", resolved, errors)
    }
    
    log.Println("")
    log.Println("Instructions post-résolution:")
    log.Println("1. Vérifiez vos vues Fauxton pour confirmer la résolution")
    log.Println("2. Testez la cohérence avec: ./monitor -database " + *database)
    log.Println("3. Les conflits résolus apparaîtront dans vos vues d'audit")
}