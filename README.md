# vens-demo

La même image, deux contextes de déploiement, deux verdicts de build.

Le workflow scanne `python:3.12-slim` **une seule fois** avec Trivy, puis passe le
rapport tel quel à deux jobs identiques. Le seul écart entre eux est le fichier de
contexte :

| Job | Contexte | Le service |
|---|---|---|
| Application exposée | [`contexts/internet-facing.yaml`](contexts/internet-facing.yaml) | API de paiement, frontale internet, sans WAF, données PCI-DSS |
| Application interne | [`contexts/internal.yaml`](contexts/internal.yaml) | Batch de nuit, sous-réseau isolé, aucun port en écoute |

Le seuil (`fail-on-severity: high`) est **identique des deux côtés**. Il n'est pas
déplacé d'un job à l'autre : si on le bougeait, la démo ne prouverait rien.

[vens](https://github.com/venslabs/vens) renote chaque CVE avec ce contexte et
sort un VEX CycloneDX ; le build casse sur ce score contextuel, pas sur le CVSS.

## Le rejouer chez vous

```
gh repo fork venslabs/vens-demo --clone
gh secret set OPENAI_API_KEY --repo <votre-fork>
gh workflow run demo --repo <votre-fork>
```

Chaque run publie le rapport de scan, les deux VEX et les deux rapports enrichis
en artefacts, et affiche la distribution des sévérités OWASP dans son résumé.
