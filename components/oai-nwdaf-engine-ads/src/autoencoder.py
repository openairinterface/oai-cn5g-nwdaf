# SPDX-License-Identifier: LicenseRef-CSSL-1.0

# Inference for the Keras LSTM autoencoder in plain numpy.
#
# The model is ~10k parameters; loading TensorFlow to run it cost the image
# 1.5 GB. This reads the same Keras .h5 file (layer config + weights) and runs
# the forward pass for the layer types the autoencoder uses: LSTM,
# RepeatVector and TimeDistributed(Dense). A retrained model built from those
# layers loads unchanged; any other layer type is rejected at start-up.

import json
import pickle

import h5py
import numpy as np


def _sigmoid(x):
    # 1 / (1 + e^-x), written so large |x| (raw byte volumes) cannot overflow
    return np.exp(-np.logaddexp(0.0, -x))


_ACTIVATIONS = {
    'tanh': np.tanh,
    'sigmoid': _sigmoid,
    'relu': lambda x: np.maximum(x, 0.0),
    'linear': lambda x: x,
}


def _lstm(x, kernel, recurrent_kernel, bias, cfg):
    """Keras LSTM, gate order i, f, c, o. x: (batch, time, features)."""
    act = _ACTIVATIONS[cfg['activation']]
    rec_act = _ACTIVATIONS[cfg['recurrent_activation']]
    units = recurrent_kernel.shape[0]
    h = np.zeros((x.shape[0], units))
    c = np.zeros((x.shape[0], units))
    outputs = []
    steps = range(x.shape[1])
    if cfg.get('go_backwards'):
        steps = reversed(steps)
    for t in steps:
        z = x[:, t, :] @ kernel + h @ recurrent_kernel + bias
        i = rec_act(z[:, :units])
        f = rec_act(z[:, units:2 * units])
        g = act(z[:, 2 * units:3 * units])
        o = rec_act(z[:, 3 * units:])
        c = f * c + i * g
        h = o * act(c)
        outputs.append(h)
    return np.stack(outputs, axis=1) if cfg['return_sequences'] else h


class KerasAutoencoder:
    def __init__(self, path):
        with h5py.File(path, 'r') as f:
            model_config = f.attrs['model_config']
            if isinstance(model_config, bytes):
                model_config = model_config.decode()
            layers = json.loads(model_config)['config']['layers']
            weights = f['model_weights']
            self.layers = []
            for layer in layers:
                kind, cfg = layer['class_name'], layer['config']
                if kind == 'InputLayer':
                    continue
                if kind not in ('LSTM', 'RepeatVector', 'TimeDistributed'):
                    raise ValueError('unsupported layer type: ' + kind)
                arrays = {}
                weights[cfg['name']].visititems(
                    lambda name, obj: arrays.__setitem__(
                        name.rsplit('/', 1)[-1].split(':')[0], obj[()])
                    if isinstance(obj, h5py.Dataset) else None)
                if kind == 'TimeDistributed' and cfg['layer']['class_name'] != 'Dense':
                    raise ValueError('unsupported TimeDistributed layer: ' +
                                     cfg['layer']['class_name'])
                self.layers.append((kind, cfg, arrays))

    def predict(self, x):
        x = np.asarray(x, dtype=np.float64)
        for kind, cfg, w in self.layers:
            if kind == 'LSTM':
                x = _lstm(x, w['kernel'], w['recurrent_kernel'],
                          w.get('bias', 0.0), cfg)
            elif kind == 'RepeatVector':
                x = np.repeat(x[:, np.newaxis, :], cfg['n'], axis=1)
            else:  # TimeDistributed(Dense)
                act = _ACTIVATIONS[cfg['layer']['config']['activation']]
                x = act(x @ w['kernel'] + w.get('bias', 0.0))
        return x


class StandardScaler:
    """The fitted sklearn StandardScaler, without sklearn: (x - mean_) / scale_."""

    def __setstate__(self, state):
        self.__dict__.update(state)

    def transform(self, x):
        x = np.asarray(x, dtype=np.float64)
        if getattr(self, 'with_mean', True):
            x = x - self.mean_
        if getattr(self, 'with_std', True):
            x = x / self.scale_
        return x


class _ScalerUnpickler(pickle.Unpickler):
    def find_class(self, module, name):
        if module.startswith('sklearn.') and name == 'StandardScaler':
            return StandardScaler
        return super().find_class(module, name)


def load_scaler(path):
    with open(path, 'rb') as f:
        return _ScalerUnpickler(f).load()
