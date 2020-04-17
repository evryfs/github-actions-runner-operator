package com.tietoevry.fss.garo.crd

import com.fasterxml.jackson.databind.annotation.JsonDeserialize
import io.fabric8.kubernetes.api.model.KubernetesResource

@JsonDeserialize
class ActionRunnerStatus: KubernetesResource