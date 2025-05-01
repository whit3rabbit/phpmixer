# Statement Shuffling Example

This example demonstrates the statement shuffling obfuscation technique, which reorders independent statements within code blocks while preserving program functionality.

## How It Works

The statement shuffling obfuscator:

1. Analyzes the PHP code to identify "chunks" of statements that can be safely reordered
2. Shuffles these chunks while preserving dependencies
3. Maintains the integrity of control flow statements and return statements

## Configuration

To enable statement shuffling in your config.yaml file:

```yaml
# Hierarchical format (recommended)
obfuscation:
  statement_shuffling:
    enabled: true                # Shuffle independent statements within blocks
    min_chunk_size: 2            # Minimum number of statements to consider for shuffling
    chunk_mode: "fixed"          # "fixed" or "ratio"
    chunk_ratio: 20              # Percentage of statements to include in a chunk

# Legacy flat-style settings
shuffle_stmts: true
shuffle_stmts_min_chunk_size: 2
shuffle_stmts_chunk_mode: "fixed"
shuffle_stmts_chunk_ratio: 20
```

## Example

The `original.php` file contains code with many independent statements that can be shuffled. After obfuscation with statement shuffling enabled, the resulting `obfuscated.php` file will have the same functionality but with a different order of statements.

To run this example:

```
cd examples/statement_shuffling
go run ../../cmd/go-php-obfuscator/main.go obfuscate file original.php -o obfuscated.php
```

## Before and After

### Before Obfuscation
```php
$a = 1;
$b = 2;
$c = 3;
```

### After Obfuscation
```php
$c = 3;
$a = 1;
$b = 2;
```

Note that while the order of statements has changed, the functionality remains identical because these statements don't depend on each other. 