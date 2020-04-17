package com.tietoevry.fss.garo.crd

import com.fasterxml.jackson.databind.annotation.JsonDeserialize
import io.fabric8.kubernetes.api.model.KubernetesResource
import io.fabric8.kubernetes.api.model.PodSpec

@JsonDeserialize
data class ActionRunnerSpec(val organization: String = "",
                            val minRunners: Int = 0,
                            val maxRunners: Int = 0,
                            val podSpec: PodSpec = PodSpec()): KubernetesResource