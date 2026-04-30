<?php

declare(strict_types=1);

namespace Fixtures\TypeError;

/**
 * Type-error fixture — return type mismatch.
 *
 * The annotated return type is int but the implementation returns a string.
 * PHPStan emits Method::return.type / similar; Psalm emits InvalidReturnType.
 *
 *   $service = new BrokenService();
 *   $service->compute(1, 2); // PHPStan/Psalm flag the wrong return
 */
final class BrokenService
{
    public function compute(int $_left, int $_right): int
    {
        return 'not-an-int';
    }
}
