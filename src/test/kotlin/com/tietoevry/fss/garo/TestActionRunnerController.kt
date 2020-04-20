package com.tietoevry.fss.garo

import io.quarkus.test.junit.QuarkusTest
import org.junit.jupiter.api.Assertions
import org.junit.jupiter.api.Disabled
import org.junit.jupiter.api.Test
import java.util.concurrent.TimeUnit
import javax.inject.Inject

/**
 * Easier way to lunch in instead of all the quarkus-crap
 */
@QuarkusTest
@Disabled
class TestActionRunnerController {

    @Inject
    lateinit var controller: ActionRunnerController

    @Test
    fun test() {
        Assertions.assertNotNull(controller)
        while (true) {
            TimeUnit.SECONDS.sleep(1)
        }
    }
}