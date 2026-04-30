<?php

declare(strict_types=1);

namespace Fixtures\Taint;

use PDO;

/**
 * Taint-flow fixture — user input flows directly into a SQL query without
 * sanitisation. Psalm taint analysis (--taint-analysis) flags this with
 * TaintedSql / TaintedInput on the $_GET source → query() sink path.
 *
 *   $repo = new TaintedQuery($pdo);
 *   $repo->fetchById(); // tainted: $_GET['id'] reaches PDO::query unsanitised
 */
final class TaintedQuery
{
    public function __construct(private readonly PDO $pdo) {}

    public function fetchById(): mixed
    {
        $id = $_GET['id'];

        return $this->pdo->query("SELECT * FROM users WHERE id = {$id}")->fetch();
    }
}
