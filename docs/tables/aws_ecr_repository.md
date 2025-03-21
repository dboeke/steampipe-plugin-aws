# Table: aws_ecr_repository

Amazon Elastic Container Registry (Amazon ECR) is a managed container image registry service.

## Examples

### Basic info

```sql
select
  repository_name,
  registry_id,
  arn,
  repository_uri,
  created_at,
  region,
  account_id
from
  aws_ecr_repository;
```


### List repositories which are not using Customer Managed Keys (CMK) for encryption

```sql
select
  repository_name,
  encryption_configuration ->> 'EncryptionType' as encryption_type,
  encryption_configuration ->> 'KmsKey' as kms_key
from
  aws_ecr_repository
where
  encryption_configuration ->> 'EncryptionType' = 'AES256';
```


### List repositories with automatic image scanning disabled

```sql
select
  repository_name,
  image_scanning_configuration ->> 'ScanOnPush' as scan_on_push
from
  aws_ecr_repository
where
  image_scanning_configuration ->> 'ScanOnPush' = 'false';
```


### List repositories whose image scanning has failed

```sql
select
  repository_name,
  detail -> 'ImageScanStatus' ->> 'Status' as scan_status
from
  aws_ecr_repository,
  jsonb_array_elements(image_details) as details,
  jsonb(details) as detail
where
  detail -> 'ImageScanStatus' ->> 'Status' = 'FAILED';
```


### List repositories whose tag immutability is disabled

```sql
select
  repository_name,
  image_tag_mutability
from
  aws_ecr_repository
where
  image_tag_mutability = 'IMMUTABLE';
```


### List repositories whose lifecycle policy rule is not configured to remove untagged and old images

```sql
select
  repository_name,
  r -> 'selection' ->> 'tagStatus' as tag_status,
  r -> 'selection' ->> 'countType' as count_type
from
  aws_ecr_repository,
  jsonb_array_elements(lifecycle_policy -> 'rules') as r
where
  (
    (r -> 'selection' ->> 'tagStatus' <> 'untagged')
    and (
      r -> 'selection' ->> 'countType' <> 'sinceImagePushed'
    )
  );
```


### List repository policy statements that grant full access for each repository

```sql
select
  title,
  p as principal,
  a as action,
  s ->> 'Effect' as effect,
  s -> 'Condition' as conditions
from
  aws_ecr_repository,
  jsonb_array_elements(policy -> 'Statement') as s,
  jsonb_array_elements_text(s -> 'Principal' -> 'AWS') as p,
  jsonb_array_elements_text(s -> 'Action') as a
where
  s ->> 'Effect' = 'Allow'
  and a in ('*', 'ecr:*');
```


### Get repository image vulnerability count by severity for each repository

```sql
select
  repository_name,
  detail -> 'ImageScanFindingsSummary' -> 'FindingSeverityCounts' ->> 'INFORMATIONAL' as informational_severity_counts,
  detail -> 'ImageScanFindingsSummary' -> 'FindingSeverityCounts' ->> 'LOW' as low_severity_counts,
  detail -> 'ImageScanFindingsSummary' -> 'FindingSeverityCounts' ->> 'MEDIUM' as medium_severity_counts
from
  aws_ecr_repository,
  jsonb_array_elements(image_details) as details,
  jsonb(details) as detail;
```
