# Guide de complétion : pkg/conflict/resolver.go

## Vue d'ensemble

Le fichier `resolver.go` contient la logique de résolution automatique
des conflits de réplication CouchDB. Chaque équipe doit compléter une
partie spécifique du code.

## Répartition des tâches

### ÉQUIPE A et B : Comparaison et suppression

**Votre mission** : Comparer les deux timestamps et supprimer la
révision la plus ancienne.

**Lignes à compléter** : \~89-109

#### TODO - 1 : Condition de comparaison

    // Ligne ~92
    if // TODO ÉQUIPE B: condition pour vérifier si conflictTime est plus récent que mainTime

#### TODO - 2 : Supprimer la révision principale

    // Ligne ~96
    _, err := // TODO ÉQUIPE B: appel à Delete() pour supprimer la révision principale (mainDoc.Rev)

#### TODO - 3 : Supprimer la révision conflictuelle

    // Ligne ~104
    _, err := // TODO ÉQUIPE B: appel à Delete() pour supprimer la révision conflictuelle (conflictRev)

## Test de votre code

### 1. Recompiler après modification

    # Windows
    go build -o bin\conflict-resolver.exe .\cmd\conflict-resolver

    # Linux/macOS
    go build -o bin/conflict-resolver ./cmd/conflict-resolver

### 2. Créer un conflit de test

    # Exécuter le scénario de votre équipe
    .\bin\scenario-a.exe    # Équipe A
    .\bin\scenario-b.exe    # Équipe B

### 3. Lister les conflits

    # Équipe A
    .\bin\conflict-resolver.exe -database ecommerce_orders -list

    # Équipe B
    .\bin\conflict-resolver.exe -database ecommerce_leads -list

### 4. Résoudre le conflit

    # Équipe A
    .\bin\conflict-resolver.exe -database ecommerce_orders -doc test_order_conflict

    # Équipe B
    .\bin\conflict-resolver.exe -database ecommerce_leads -doc lead_sync_conflict

### 5. Vérifier la résolution

    # Re-lister pour confirmer qu'il n'y a plus de conflits
    .\bin\conflict-resolver.exe -database ecommerce_orders -list

## Critères de validation

✅ **Votre code est correct si** : - La compilation réussit sans
erreur - Les conflits sont détectés par `-``list` - La résolution
choisit la révision la plus récente - Après résolution, `-``list` ne
montre plus de conflits - Les logs indiquent clairement quelle révision
a été supprimée

❌ **Erreurs communes** : - Oublier l'assertion de type `.(string)` -
Utiliser le mauvais format de temps (doit être `time.RFC3339`) -
Inverser la logique de comparaison (garder l'ancienne au lieu de la
récente) - Oublier le `?``rev``=` dans l'URL de suppression

------------------------------------------------------------------------

## Comprendre la logique métier

### Pourquoi garder la version la plus récente ?

Dans un registre distribué e-commerce : - **Principe** : "Last Write
Wins" (dernière écriture gagne) - **Justification** : La version la plus
récente contient généralement les informations les plus à jour -
**Exception** : Dans certains cas, on pourrait vouloir une stratégie
différente (ex: garder la version avec le montant le plus élevé)

### Scénario typique

1.  **T0** : Document créé sur NA1 (`rev``: 1-abc`)
2.  **T1** : Répliqué sur NA2 (`rev``: 1-abc`)
3.  **T2** : NA2 tombe en panne
4.  **T3** : Modification sur NA1 (`rev``: 2-def`, timestamp: 23:53:25)
5.  **T4** : NA2 redémarre et crée une modification concurrente
    (`rev``: 2-xyz`, timestamp: 23:53:12)
6.  **T5** : Conflit détecté lors de la réplication
7.  **T6** : Résolution automatique → Garder `2-def` (plus récent),
    supprimer `2-xyz`

## Aide et ressources

- Documentation Go Time : https://pkg.go.dev/time
- Documentation CouchDB Conflicts :
  https://docs.couchdb.org/en/stable/replication/conflicts.html
- Vos enseignants pour toute question !

Bon courage ! 🚀

