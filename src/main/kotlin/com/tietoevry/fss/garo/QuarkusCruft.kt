package com.tietoevry.fss.garo

import io.quarkus.runtime.StartupEvent
import javax.enterprise.context.ApplicationScoped
import javax.enterprise.event.Observes

@ApplicationScoped
class QuarkusCruft {

    fun onStart(@Observes startupEvent: StartupEvent?, actionRunnerController: ActionRunnerController) {
        actionRunnerController.toString()
    }
}