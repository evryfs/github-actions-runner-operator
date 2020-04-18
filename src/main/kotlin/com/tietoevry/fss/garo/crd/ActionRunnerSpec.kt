package com.tietoevry.fss.garo.crd

import com.fasterxml.jackson.databind.annotation.JsonDeserialize
import io.fabric8.kubernetes.api.model.KubernetesResource
import io.fabric8.kubernetes.api.model.PodSpec
import io.fabric8.kubernetes.api.model.SecretKeySelector

@JsonDeserialize
data class ActionRunnerSpec(val organization: String = "",
                            val tokenRef: SecretKeySelector = SecretKeySelector(),
                            val minRunners: Int = 0,
                            val maxRunners: Int = 0,
                            val podSpec: PodSpec = PodSpec()): KubernetesResource