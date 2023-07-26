import z3
from enum import Enum


class Policies(Enum):
    Delay = 1
    Count = 2
    GetHeader = 3
    SetHeader = 4
    SetDeadline = 5


# Define the policy constraints -- 1 implies only on sender.
policy_constraints = {
    Policies.SetDeadline: 1,
}


# Define the application graph
def define_appl_graph():
    nodes = ["A", "B", "C", "D", "E", "F", "G"]
    edges = {
        "A": ["B", "C"],
        "B": ["E"],
        "C": ["D"],
        "D": ["E"],
        "E": ["F", "G"]
    }
    return nodes, edges


# TODO: Handle multiple functions in a single policy.
def define_user_policies():
    policies = [{
        "context": ["A", "*"],
        "function": Policies.SetHeader
    }, {
        "context": ["*", "F"],
        "function": Policies.Count
    }, {
        "context": ["A", "*", "E", "*"],
        "function": Policies.SetDeadline
    }]
    return policies


# Start from a node and get all paths that satisfy the policy context.
def forward_policy_context(policy_context, appl_edges):
    curr_context_list = [[policy_context[0]]]
    prev_node = policy_context[0]
    for i in range(1, len(policy_context)):
        if policy_context[i] != "*":
            prev_node = policy_context[i]
            for context in curr_context_list:
                context.append(policy_context[i])
        else:
            target_node = policy_context[
                i + 1] if i < len(policy_context) - 1 else None
            if target_node:
                bft_queue = [[n] for n in appl_edges[prev_node]]

                # Keep track of paths from previous node to target node.
                new_context_list = []
                while bft_queue:
                    # print("BFT Queue: ", bft_queue)
                    # print("Current Context List: ", curr_context_list)
                    curr_path = bft_queue.pop(0)
                    curr_node = curr_path[-1]

                    # Unroll BFS until target node is found or a leaf is met.
                    if curr_node == target_node:
                        for context in curr_context_list:
                            new_context_list.append(context + curr_path[:-1])
                    elif curr_node in appl_edges:
                        # Add the children to bft queue.
                        for n in appl_edges[curr_node]:
                            bft_queue.append(curr_path + [n])
                curr_context_list = new_context_list
            else:
                # If target node is None, then just add all the children of the previous node.
                new_context_list = []
                for context in curr_context_list:
                    for n in appl_edges[prev_node]:
                        new_context_list.append(context + [n])
                curr_context_list = new_context_list

    return curr_context_list


# Get all paths that satisfy the policy context and end on the given policy_context.
def backward_policy_context(target_node, appl_edges):
    # Construct the parent-child edges from appl_edges.
    parents = {}
    for n, e in appl_edges.items():
        for c in e:
            if c in parents:
                parents[c].append(n)
            else:
                parents[c] = [n]
    print(parents)

    backward_bft_queue = [[target_node]]
    context_list = []
    while backward_bft_queue:
        # print("BFT Queue: ", bft_queue)
        # print("Current Context List: ", curr_context_list)
        curr_path = backward_bft_queue.pop(0)
        curr_node = curr_path[0]
        if len(curr_path) > 1:
            context_list.append(curr_path)

        # Unroll BFS until a root node is met.
        if curr_node in parents:
            # Add the children to bft queue.
            for n in parents[curr_node]:
                backward_bft_queue.append([n] + curr_path)

    return context_list


# Expand the policy context to get all possible request contexts.
def expand_policy_context(policy_context, appl_edges):
    if policy_context[0] != "*":
        return forward_policy_context(policy_context, appl_edges)
    else:
        pre_context_list = backward_policy_context(policy_context[1],
                                                   appl_edges)
        post_context_list = forward_policy_context(policy_context[1:],
                                                   appl_edges)
        context_list = []
        for pre_context in pre_context_list:
            if post_context_list:
                for post_context in post_context_list:
                    context_list.append(pre_context + post_context[1:])
            else:
                context_list.append(pre_context)
        return context_list


# TODO: Only considers a single variable for * operator.
def get_policy_impls(policy_context, appl_edges):
    pen_set = []
    ult_set = []
    ultimate_node = policy_context[-1]
    if ultimate_node == "*":
        # TODO: Handle set of nodes.
        penultimate_node = policy_context[-2]
        pen_set.append(penultimate_node)
        for n in appl_edges[penultimate_node]:
            ult_set.append(n)
    else:
        ult_set.append(ultimate_node)
        for n, e in appl_edges.items():
            if ultimate_node in e:
                pen_set.append(n)

    return pen_set, ult_set


def main():
    # Define the application graph
    appl_nodes, appl_edges = define_appl_graph()
    # Define the user policies
    user_policies = define_user_policies()

    # Define the objective
    objective = 2

    # Expand the policy contexts
    all_req_contexts = {}
    for i in range(len(user_policies)):
        policy = user_policies[i]
        req_contexts = expand_policy_context(policy["context"], appl_edges)
        print("Expanding policy context ", policy["context"])
        print("Expanded contexts: ", req_contexts)

        for req_context in req_contexts:
            if tuple(req_context) in all_req_contexts:
                all_req_contexts[tuple(req_context)].append(i)
            else:
                all_req_contexts[tuple(req_context)] = [i]
    print(all_req_contexts)

    # Define the variables
    all_req_contexts_list = list(all_req_contexts.keys())
    num_req_contexts = len(all_req_contexts_list)
    num_policies = len(user_policies)
    num_nodes = len(appl_nodes)

    # Define the "Belong to the policy context" variables
    B = [[z3.Bool("b_{}_{}".format(i, j)) for j in range(num_policies)]
         for i in range(num_req_contexts)]
    # print(B)

    # Define the "Implements" variables
    I = [[z3.Bool("i_{}_{}".format(m, j)) for j in range(num_policies)]
         for m in range(num_nodes)]
    # print(I)

    # Define the "Exists" variables
    X = [z3.Bool("x_{}".format(m)) for m in range(num_nodes)]
    # print(X)

    # Define the "Executes" variables
    E = [[[z3.Bool("e_{}_{}_{}".format(i, j, m)) for m in range(num_nodes)]
          for j in range(num_policies)] for i in range(num_req_contexts)]
    # print(E)

    # Define the solver
    o = z3.Optimize()

    # Add belonging constraints
    for i in range(num_req_contexts):
        req_context = all_req_contexts_list[i]
        for j in range(num_policies):
            if j in all_req_contexts[tuple(req_context)]:
                o.add(B[i][j])
            else:
                o.add(z3.Not(B[i][j]))

    # Add the bi-implication constraints
    for i in range(num_req_contexts):
        for j in range(num_policies):
            alpha = B[i][j]
            beta = z3.BoolVal(False)
            for m in range(num_nodes):
                beta = z3.Or(z3.And(z3.And(E[i][j][m], I[m][j]), X[m]), beta)
            o.add(z3.Implies(alpha, beta))
            o.add(z3.Implies(beta, alpha))

    # Add the request context constraints
    for i in range(num_req_contexts):
        req_context = all_req_contexts_list[i]
        for j in range(num_policies):
            for m in range(num_nodes):
                if appl_nodes[m] not in req_context:
                    o.add(z3.Not(E[i][j][m]))

    # Add the policy constraints
    for j in range(num_policies):
        policy = user_policies[j]
        pen_set, ult_set = get_policy_impls(policy["context"], appl_edges)
        pen_set = [appl_nodes.index(n) for n in pen_set]
        ult_set = [appl_nodes.index(n) for n in ult_set]
        print("Possible impl for policy:", policy, pen_set, ult_set)
        alpha = z3.BoolVal(True)
        for m in pen_set:
            alpha = z3.And(I[m][j], alpha)
        beta = z3.BoolVal(True)
        for m in ult_set:
            beta = z3.And(I[m][j], beta)
        o.add(z3.Xor(alpha, beta))

        valid = pen_set
        if policy["function"] not in policy_constraints:
            valid.extend(ult_set)
        print("Valid: ", valid)

        # Add the constraint for invalid implementations
        for m in range(num_nodes):
            if m not in valid:
                o.add(z3.Not(I[m][j]))

    # Exactly one node executes a policy for a given context
    # CHECK: Is this constraint correct?
    for i in range(num_req_contexts):
        for j in range(num_policies):
            # If B[i][j] is true, then exactly one node executes the policy
            # If B[i][j] is false, then no node executes the policy
            alpha_ij = z3.Sum([
                z3.If(E[i][j][m], z3.IntVal(1), z3.IntVal(0))
                for m in range(num_nodes)
            ])
            o.add(z3.If(B[i][j], alpha_ij == 1, alpha_ij == 0))

    # Define the objective
    num_sidecars = z3.Sum(
        [z3.If(X[m], z3.IntVal(1), z3.IntVal(0)) for m in range(num_nodes)])
    # o.add(num_sidecars <= objective)

    # Check if the constraints are satisfiable
    o.minimize(num_sidecars)
    sat = o.check()
    print(sat)

    if sat == z3.sat:
        model = o.model()

        # Get the X[m] values for the solution
        sidecars = []
        for m in range(num_nodes):
            if model.evaluate(X[m]):
                sidecars.append(appl_nodes[m])
        print(sidecars)

        # Get the I[m][j] values for the solution
        placements = []
        for m in range(num_nodes):
            for j in range(num_policies):
                if model.evaluate(I[m][j]) and model.evaluate(X[m]):
                    placements.append((appl_nodes[m], j))
        # print(placements)

        # Get the E[i][j][m] values for the solution
        executions = []
        for i in range(num_req_contexts):
            for j in range(num_policies):
                for m in range(num_nodes):
                    if model.evaluate(E[i][j][m]):
                        executions.append(
                            (all_req_contexts_list[i], j, appl_nodes[m]))
        for execution in executions:
            print(execution)


if __name__ == "__main__":
    main()