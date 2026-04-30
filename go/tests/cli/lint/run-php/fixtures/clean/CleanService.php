<?php

declare(strict_types=1);

namespace Fixtures\Clean;

/**
 * Clean fixture — passes both PHPStan and Psalm with no findings.
 *
 *   $service = new CleanService();
 *   $service->add(2, 3); // → 5
 */
final class CleanService
{
    public function add(int $left, int $right): int
    {
        return $left + $right;
    }
}
